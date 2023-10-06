package servers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v2"

	"github.com/yuhaohwang/bililive-go/src/configs"
	"github.com/yuhaohwang/bililive-go/src/consts"
	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/listeners"
	"github.com/yuhaohwang/bililive-go/src/live"
	"github.com/yuhaohwang/bililive-go/src/pushers"
	"github.com/yuhaohwang/bililive-go/src/recorders"
)

// parseInfo 从直播信息对象中提取相关数据并构建一个 live.Info 结构。
func parseInfo(ctx context.Context, l live.Live) *live.Info {
	// 获取应用程序实例
	inst := instance.GetInstance(ctx)

	// 从缓存中获取直播信息对象
	obj, _ := inst.Cache.Get(l)
	info := obj.(*live.Info)

	// 获取直播的原始 URL
	live_url := l.GetRawUrl()

	// 使用 URL 从配置中获取房间信息
	room, err := inst.Config.GetLiveRoomByUrl(live_url)
	if err == nil {
		info.Listen = room.Listen
		info.Record = room.Record
		info.Push = room.Push
		info.RtmpUrl = room.Rtmp
	}

	// 检查是否有监听器和录制器，并将结果存储在相应的字段中
	info.Listening = inst.ListenerManager.(listeners.Manager).HasListener(ctx, l.GetLiveId())
	info.Recording = inst.RecorderManager.(recorders.Manager).HasRecorder(ctx, l.GetLiveId())
	info.Pushing = inst.PusherManager.(pushers.Manager).HasPusher(ctx, l.GetLiveId())

	// 返回填充好数据的 live.Info 结构
	return info
}

// 获取所有直播信息
func getAllLives(writer http.ResponseWriter, r *http.Request) {
	// 获取应用程序实例
	inst := instance.GetInstance(r.Context())
	// 创建直播信息切片
	lives := liveSlice(make([]*live.Info, 0, 4))
	// 遍历所有直播
	for _, v := range inst.Lives {
		// 解析直播信息并添加到切片中
		lives = append(lives, parseInfo(r.Context(), v))
	}

	// 按某个标准排序直播信息切片
	sort.Sort(lives)
	// 返回 JSON 格式的直播信息切片
	writeJSON(writer, lives)
}

// 获取单个直播信息
func getLive(writer http.ResponseWriter, r *http.Request) {
	// 获取应用程序实例
	inst := instance.GetInstance(r.Context())
	// 获取请求中的直播 ID
	vars := mux.Vars(r)
	// 根据直播 ID 查找直播
	live, ok := inst.Lives[live.ID(vars["id"])]
	if !ok {
		// 直播不存在，返回错误响应
		writeJsonWithStatusCode(writer, http.StatusNotFound, commonResp{
			ErrNo:  http.StatusNotFound,
			ErrMsg: fmt.Sprintf("live id: %s 找不到", vars["id"]),
		})
		return
	}
	// 解析直播信息并返回
	writeJSON(writer, parseInfo(r.Context(), live))
}

/*
Post 数据示例

[

	{
		"url": "http://live.bilibili.com/1030",
		"listen": true
	},
	{
		"url": "https://live.bilibili.com/493",
		"listen": true
	}

]
*/
func addLives(writer http.ResponseWriter, r *http.Request) {
	// 读取请求的数据
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(writer, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	// 获取应用程序实例
	inst := instance.GetInstance(r.Context())
	// 创建直播信息切片
	info := liveSlice(make([]*live.Info, 0))
	// 创建错误消息切片
	errorMessages := make([]string, 0, 4)
	// 遍历请求中的直播信息
	gjson.ParseBytes(b).ForEach(func(key, value gjson.Result) bool {
		urlStr := strings.Trim(value.Get("url").String(), " ")
		isListen := value.Get("listen").Bool()
		isRecord := value.Get("record").Bool()
		rtmpStr := strings.Trim(value.Get("rtmp").String(), " ")
		isPush := value.Get("push").Bool()
		// 调用添加直播信息的实现函数
		if retInfo, err := addLiveImpl(r.Context(), urlStr, isListen, isRecord, rtmpStr, isPush); err != nil {
			msg := urlStr + "：" + err.Error()
			inst.Logger.Error(msg)
			errorMessages = append(errorMessages, msg)
			return true
		} else {
			info = append(info, retInfo)
		}
		return true
	})
	// 按某个标准排序直播信息切片
	sort.Sort(info)
	// TODO：返回错误消息
	writeJSON(writer, info)
}

// 添加直播信息的实现函数
func addLiveImpl(ctx context.Context, urlStr string, isListen bool, isRecord bool, rtmpStr string, isPush bool) (info *live.Info, err error) {
	// 如果 URL 不以 "http://" 或 "https://" 开头，则添加 "https://" 前缀
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}
	// 解析 URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, errors.New("无法解析 URL：" + urlStr)
	}

	// 获取应用程序实例
	inst := instance.GetInstance(ctx)
	opts := make([]live.Option, 0)
	// 如果存在与主机匹配的 Cookie，则添加到选项中
	if v, ok := inst.Config.Cookies[u.Host]; ok {
		opts = append(opts, live.WithKVStringCookies(u, v))
	}
	// 创建新的直播实例
	newLive, err := live.New(u, inst.Cache, opts...)
	if err != nil {
		return nil, err
	}
	// 如果直播信息尚未存在于应用程序中，则添加
	if _, ok := inst.Lives[newLive.GetLiveId()]; !ok {
		inst.Lives[newLive.GetLiveId()] = newLive
		if isListen {
			inst.ListenerManager.(listeners.Manager).AddListener(ctx, newLive)
		}
		info = parseInfo(ctx, newLive)

		info.Listen = isListen
		info.Record = isRecord

		// 如果rtmp不为空则添加相关字段
		if rtmpStr != "" {
			// 如果 RTMP 不以 "rtmp://" 或 "rtmps://" 开头，则添加 "rtmp://" 前缀
			if !strings.HasPrefix(rtmpStr, "rtmp://") && !strings.HasPrefix(rtmpStr, "rtmps://") {
				rtmpStr = "rtmp://" + rtmpStr
			}
			// 解析 RTMP
			r, err := url.Parse(rtmpStr)
			if err != nil {
				return nil, errors.New("无法解析 RTMP：" + rtmpStr)
			}
			info.RtmpUrl = r.String()
			info.Push = isPush
		}

		liveRoom := configs.LiveRoom{
			LiveId: newLive.GetLiveId(),
			Url:    u.String(),
			Listen: isListen,
			Record: isRecord,
			Rtmp:   rtmpStr,
			Push:   isPush,
		}
		inst.Config.LiveRooms = append(inst.Config.LiveRooms, liveRoom)
	}
	return info, nil
}

// 移除直播信息
func removeLive(writer http.ResponseWriter, r *http.Request) {
	// 获取应用程序实例
	inst := instance.GetInstance(r.Context())
	// 获取请求中的直播 ID
	vars := mux.Vars(r)
	// 根据直播 ID 查找直播
	live, ok := inst.Lives[live.ID(vars["id"])]
	if !ok {
		// 直播不存在，返回错误响应
		writeJsonWithStatusCode(writer, http.StatusNotFound, commonResp{
			ErrNo:  http.StatusNotFound,
			ErrMsg: fmt.Sprintf("live id: %s 找不到", vars["id"]),
		})
		return
	}
	// 调用移除直播信息的实现函数
	if err := removeLiveImpl(r.Context(), live); err != nil {
		writeJsonWithStatusCode(writer, http.StatusBadRequest, commonResp{
			ErrNo:  http.StatusBadRequest,
			ErrMsg: err.Error(),
		})
		return
	}
	// 返回成功响应
	writeJSON(writer, commonResp{
		Data: "OK",
	})
}

// 移除直播信息的实现函数
func removeLiveImpl(ctx context.Context, live live.Live) error {
	// 获取应用程序实例
	inst := instance.GetInstance(ctx)
	lm := inst.ListenerManager.(listeners.Manager)
	// 如果有监听器，则停止监听
	if lm.HasListener(ctx, live.GetLiveId()) {
		if err := lm.RemoveListener(ctx, live.GetLiveId()); err != nil {
			return err
		}
	}
	// 从应用程序中移除直播信息
	delete(inst.Lives, live.GetLiveId())
	// 从配置中移除直播房间信息
	inst.Config.RemoveLiveRoomByUrl(live.GetRawUrl())
	return nil
}

// 获取配置信息
func getConfig(writer http.ResponseWriter, r *http.Request) {
	// 返回应用程序配置信息
	writeJSON(writer, instance.GetInstance(r.Context()).Config)
}

// 更新配置信息
func putConfig(writer http.ResponseWriter, r *http.Request) {
	// 获取应用程序实例
	config := instance.GetInstance(r.Context()).Config
	// 刷新直播房间索引缓存
	config.RefreshLiveRoomIndexCache()
	// 将配置信息持久化到文件
	if err := config.Marshal(); err != nil {
		writeJsonWithStatusCode(writer, http.StatusBadRequest, commonResp{
			ErrNo:  http.StatusBadRequest,
			ErrMsg: err.Error(),
		})
		return
	}
	// 返回成功响应
	writeJsonWithStatusCode(writer, http.StatusOK, commonResp{
		Data: "OK",
	})
}

// 获取原始配置信息
func getRawConfig(writer http.ResponseWriter, r *http.Request) {
	// 将应用程序配置信息转换为 YAML 格式并返回
	b, err := yaml.Marshal(instance.GetInstance(r.Context()).Config)
	if err != nil {
		writeJsonWithStatusCode(writer, http.StatusInternalServerError, commonResp{
			ErrNo:  http.StatusBadRequest,
			ErrMsg: err.Error(),
		})
		return
	}
	writeJSON(writer, map[string]string{
		"config": string(b),
	})
}

// 更新原始配置信息
func putRawConfig(writer http.ResponseWriter, r *http.Request) {
	// 读取请求的数据
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJsonWithStatusCode(writer, http.StatusBadRequest, commonResp{
			ErrNo:  http.StatusBadRequest,
			ErrMsg: err.Error(),
		})
		return
	}
	ctx := r.Context()
	inst := instance.GetInstance(ctx)
	var jsonBody map[string]interface{}
	json.Unmarshal(b, &jsonBody)
	configPath, err := inst.Config.GetFilePath()
	if err != nil {
		writeJsonWithStatusCode(writer, http.StatusInternalServerError, commonResp{
			ErrNo:  http.StatusInternalServerError,
			ErrMsg: err.Error(),
		})
		return
	}
	// 创建新的配置实例并应用直播房间信息
	newConfig, err := configs.NewConfigWithBytes([]byte(jsonBody["config"].(string)))
	if err != nil {
		writeJsonWithStatusCode(writer, http.StatusInternalServerError, commonResp{
			ErrNo:  http.StatusInternalServerError,
			ErrMsg: err.Error(),
		})
		return
	}
	oldConfig := inst.Config
	newConfig.File = oldConfig.File
	if err := applyLiveRoomsByConfig(ctx, newConfig.LiveRooms); err != nil {
		writeJSON(writer, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	newConfig.LiveRooms = oldConfig.LiveRooms
	// 将配置信息持久化到文件
	os.WriteFile(configPath, []byte(jsonBody["config"].(string)), os.ModePerm)
	inst.Config = newConfig
	newConfig.RefreshLiveRoomIndexCache()
	// 返回成功响应
	writeJSON(writer, commonResp{
		Data: "OK",
	})
}

// 根据新的配置应用直播房间信息
func applyLiveRoomsByConfig(ctx context.Context, newLiveRooms []configs.LiveRoom) error {
	inst := instance.GetInstance(ctx)
	currentConfig := inst.Config
	currentConfig.RefreshLiveRoomIndexCache()
	newUrlMap := make(map[string]*configs.LiveRoom)
	for _, newRoom := range newLiveRooms {
		newUrlMap[newRoom.Url] = &newRoom
		if room, err := currentConfig.GetLiveRoomByUrl(newRoom.Url); err != nil {
			// 添加直播信息
			if _, err := addLiveImpl(ctx, newRoom.Url, newRoom.Listen, newRoom.Record, newRoom.Rtmp, newRoom.Push); err != nil {
				return err
			}
		} else {
			live, ok := inst.Lives[live.ID(room.LiveId)]
			if !ok {
				return fmt.Errorf("live id: %s 找不到", room.LiveId)
			}
			if room.Listen != newRoom.Listen {
				if newRoom.Listen {
					// 开始监听
					if err := actionMap["listen"]["start"](ctx, live); err != nil {
						return err
					}
				} else {
					// 停止监听
					if err := actionMap["listen"]["stop"](ctx, live); err != nil {
						return err
					}
				}
				room.Listen = newRoom.Listen
			}
		}
	}
	loopRooms := currentConfig.LiveRooms
	for _, room := range loopRooms {
		if _, ok := newUrlMap[room.Url]; !ok {
			// 移除直播信息
			live, ok := inst.Lives[live.ID(room.LiveId)]
			if !ok {
				return fmt.Errorf("live id: %s 找不到", room.LiveId)
			}
			removeLiveImpl(ctx, live)
		}
	}
	return nil
}

// 获取应用程序信息
func getInfo(writer http.ResponseWriter, r *http.Request) {
	// 返回应用程序信息
	writeJSON(writer, consts.AppInfo)
}

// 获取文件信息
func getFileInfo(writer http.ResponseWriter, r *http.Request) {
	// 获取请求中的文件路径变量
	vars := mux.Vars(r)
	path := vars["path"]

	inst := instance.GetInstance(r.Context())
	base, err := filepath.Abs(inst.Config.OutPutPath)
	if err != nil {
		writeJSON(writer, commonResp{
			ErrMsg: "无效输出目录",
		})
		return
	}

	absPath, err := filepath.Abs(filepath.Join(base, path))
	if err != nil {
		writeJSON(writer, commonResp{
			ErrMsg: "无效路径",
		})
		return
	}
	if !strings.HasPrefix(absPath, base) {
		writeJSON(writer, commonResp{
			ErrMsg: "异常路径",
		})
		return
	}

	files, err := os.ReadDir(absPath)
	if err != nil {
		writeJSON(writer, commonResp{
			ErrMsg: "获取目录失败",
		})
		return
	}

	type jsonFile struct {
		IsFolder     bool   `json:"is_folder"`
		Name         string `json:"name"`
		LastModified int64  `json:"last_modified"`
		Size         int64  `json:"size"`
	}
	jsonFiles := make([]jsonFile, len(files))
	json := struct {
		Files []jsonFile `json:"files"`
		Path  string     `json:"path"`
	}{
		Files: jsonFiles,
		Path:  path,
	}
	for i, file := range files {
		jsonFiles[i].IsFolder = file.IsDir()
		jsonFiles[i].Name = file.Name()

		// 使用 os.Stat 获取文件的详细信息
		fileInfo, err := os.Stat(filepath.Join(absPath, file.Name()))
		if err != nil {
			writeJSON(writer, commonResp{
				ErrMsg: "获取文件信息失败",
			})
			return
		}
		jsonFiles[i].LastModified = fileInfo.ModTime().Unix()
		if !file.IsDir() {
			jsonFiles[i].Size = fileInfo.Size()
		}
	}
	json.Files = jsonFiles

	// 返回文件信息
	writeJSON(writer, json)
}

// 设置直播转推地址的实现函数
func setRtmp(writer http.ResponseWriter, r *http.Request) {
	// 读取请求的数据
	b, err := io.ReadAll(r.Body)
	if err != nil {
		writeJsonWithStatusCode(writer, http.StatusBadRequest, commonResp{
			ErrNo:  http.StatusBadRequest,
			ErrMsg: err.Error(),
		})
		return
	}

	result := gjson.ParseBytes(b)

	// 获取应用程序实例
	inst := instance.GetInstance(r.Context())
	// 获取请求中的变量
	vars := mux.Vars(r)
	resp := commonResp{}
	// 根据直播 ID 查找直播
	live, ok := inst.Lives[live.ID(vars["id"])]
	if !ok {
		// 直播不存在，返回错误响应
		resp.ErrNo = http.StatusNotFound
		resp.ErrMsg = fmt.Sprintf("live id: %s 找不到", vars["id"])
		writeJsonWithStatusCode(writer, http.StatusNotFound, resp)
		return
	}
	// 根据直播 URL 获取房间信息
	room, err := inst.Config.GetLiveRoomByUrl(live.GetRawUrl())
	if err != nil {
		resp.ErrNo = http.StatusNotFound
		resp.ErrMsg = fmt.Sprintf("房间：%s 找不到", live.GetRawUrl())
		writeJsonWithStatusCode(writer, http.StatusNotFound, resp)
		return
	}

	rtmpValue, rtmpExists := result.Get("rtmp").Value().(string)

	if !rtmpExists {
		// 处理缺少必要字段的情况
		errMsg := ""
		if !rtmpExists {
			errMsg = "rtmp参数不存在"
		}
		writeJSON(writer, commonResp{
			ErrMsg: errMsg,
		})
		return
	}

	rtmpStr := strings.Trim(rtmpValue, " ")

	// 如果 RTMP 不以 "rtmp://" 或 "rtmps://" 开头，则添加 "rtmp://" 前缀
	if !strings.HasPrefix(rtmpStr, "rtmp://") && !strings.HasPrefix(rtmpStr, "rtmps://") {
		rtmpStr = "rtmp://" + rtmpStr
	}

	room.Rtmp = rtmpStr

	// 返回成功响应
	writeJsonWithStatusCode(writer, http.StatusOK, commonResp{
		Data: "OK",
	})
}

type ActionFunc func(ctx context.Context, live live.Live) error

var actionMap = map[string]map[string]ActionFunc{
	"listen": {
		"start": func(ctx context.Context, live live.Live) error {
			inst := instance.GetInstance(ctx)
			manager, _ := inst.ListenerManager.(listeners.Manager)
			return manager.AddListener(ctx, live)
		},
		"stop": func(ctx context.Context, live live.Live) error {
			inst := instance.GetInstance(ctx)
			lm, _ := inst.ListenerManager.(listeners.Manager)
			rm, _ := inst.RecorderManager.(recorders.Manager)
			pm, _ := inst.PusherManager.(pushers.Manager)
			rm.RemoveRecorder(ctx, live.GetLiveId())
			pm.RemovePusher(ctx, live.GetLiveId())
			return lm.RemoveListener(ctx, live.GetLiveId())
		},
	},
	"record": {
		"start": func(ctx context.Context, live live.Live) error {
			inst := instance.GetInstance(ctx)
			manager, _ := inst.RecorderManager.(recorders.Manager)
			return manager.AddRecorder(ctx, live)
		},
		"stop": func(ctx context.Context, live live.Live) error {
			inst := instance.GetInstance(ctx)
			manager, _ := inst.RecorderManager.(recorders.Manager)
			return manager.RemoveRecorder(ctx, live.GetLiveId())
		},
	},
	"push": {
		"start": func(ctx context.Context, live live.Live) error {
			inst := instance.GetInstance(ctx)
			manager, _ := inst.PusherManager.(pushers.Manager)
			return manager.AddPusher(ctx, live)
		},
		"stop": func(ctx context.Context, live live.Live) error {
			inst := instance.GetInstance(ctx)
			manager, _ := inst.PusherManager.(pushers.Manager)
			return manager.RemovePusher(ctx, live.GetLiveId())
		},
	},
}

func executeAction(ctx context.Context, live live.Live, room *configs.LiveRoom, resource string, action string) error {
	setRoomStatus := func(target *bool, status bool) {
		*target = status
	}

	_, exists := actionMap[resource]

	if !exists {
		return errors.New("无效资源: " + resource)
	}

	_, exists = actionMap[resource][action]
	if !exists {
		return errors.New("无效操作: " + action)
	}

	err := errors.New("")

	switch resource {
	case "listen":
		if action == "start" {
			setRoomStatus(&room.Listen, true)
			if err = actionMap[resource][action](ctx, live); err != nil {
				setRoomStatus(&room.Listening, true)
				return err
			}
		} else {
			setRoomStatus(&room.Listen, false)
			if err = actionMap[resource][action](ctx, live); err != nil {
				setRoomStatus(&room.Listening, false)
				return err
			}
		}
	case "record":
		if action == "start" {
			if !room.Listen {
				setRoomStatus(&room.Listen, true)
				if err = actionMap["listen"][action](ctx, live); err != nil {
					setRoomStatus(&room.Listening, true)
				}
			}
			setRoomStatus(&room.Record, true)
			if err = actionMap[resource][action](ctx, live); err != nil {
				setRoomStatus(&room.Recordind, true)
				return err
			}
		} else {
			setRoomStatus(&room.Record, false)
			if err = actionMap[resource][action](ctx, live); err != nil {
				setRoomStatus(&room.Recordind, false)
				return err
			}
			if !room.Record && !room.Push {
				setRoomStatus(&room.Listen, false)
				if err = actionMap["listen"][action](ctx, live); err != nil {
					setRoomStatus(&room.Listening, false)
				}
			}
		}
	case "push":
		if action == "start" {
			if room.Rtmp == "" {
				return errors.New("RTMP地址不存在")
			}
			if !room.Listen {
				setRoomStatus(&room.Listen, true)
				if err = actionMap["listen"][action](ctx, live); err != nil {
					setRoomStatus(&room.Listening, true)
				}
			}
			setRoomStatus(&room.Push, true)
			if err = actionMap[resource][action](ctx, live); err != nil {
				setRoomStatus(&room.Pushing, true)
				return err
			}
		} else {
			setRoomStatus(&room.Push, false)
			if err = actionMap[resource][action](ctx, live); err != nil {
				setRoomStatus(&room.Pushing, false)
				return err
			}
			if !room.Record && !room.Push {
				setRoomStatus(&room.Listen, false)
				if err = actionMap["listen"][action](ctx, live); err != nil {
					setRoomStatus(&room.Listening, false)
				}
			}
		}
	}

	return err
}

func mainHandler(writer http.ResponseWriter, r *http.Request) {
	inst := instance.GetInstance(r.Context())
	vars := mux.Vars(r)
	resp := commonResp{}

	live, exists := inst.Lives[live.ID(vars["id"])]
	if !exists {
		resp.ErrNo = http.StatusBadRequest
		resp.ErrMsg = fmt.Sprintf("live id: %s 找不到", vars["id"])
		writeJsonWithStatusCode(writer, http.StatusNotFound, resp)
		return
	}

	room, err := inst.Config.GetLiveRoomByUrl(live.GetRawUrl())
	if err != nil {
		resp.ErrNo = http.StatusNotFound
		resp.ErrMsg = fmt.Sprintf("房间：%s 找不到", live.GetRawUrl())
		writeJsonWithStatusCode(writer, http.StatusNotFound, resp)
		return
	}

	resource, exists := vars["resource"]

	// 如果不存在资源字段，默认设置资源为listen字段
	if !exists {
		resource = "listen"
	}

	_, isExists := actionMap[resource]

	// 如果存在资源字段，但是无效资源
	if exists && !isExists {
		writeJsonWithStatusCode(writer, http.StatusBadRequest, fmt.Sprintf("无效资源：%s", vars["resource"]))
		return
	}

	action, exists := vars["action"]
	_, isExists = actionMap[resource]
	if !exists && !isExists {
		writeJsonWithStatusCode(writer, http.StatusBadRequest, fmt.Sprintf("无效操作：%s", vars["action"]))
		return
	}

	if err := executeAction(r.Context(), live, room, resource, action); err != nil {
		resp.ErrNo = http.StatusBadRequest
		resp.ErrMsg = err.Error()
		writeJsonWithStatusCode(writer, http.StatusBadRequest, resp)
		return
	}

	writeJSON(writer, parseInfo(r.Context(), live))
}
