package servers

import (
	"context"
	"net/http"
	_ "net/http/pprof" // 导入 net/http/pprof 包，用于性能分析

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/yuhaohwang/bililive-go/src/instance"
	"github.com/yuhaohwang/bililive-go/src/webapp"
)

const (
	apiRouterPrefix = "/api" // 定义 API 路由的前缀
)

// Server 结构体表示服务器对象。
type Server struct {
	server *http.Server
}

// initMux 函数初始化路由处理器，并添加中间件。
func initMux(ctx context.Context) *mux.Router {
	m := mux.NewRouter()
	m.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w,
				r.WithContext(
					context.WithValue(
						r.Context(),
						instance.Key,
						instance.GetInstance(ctx),
					),
				),
			)
		})
	}, log) // 使用 log 中间件记录请求日志

	// 设置 API 路由
	apiRoute := m.PathPrefix(apiRouterPrefix).Subrouter()
	apiRoute.Use(mux.CORSMethodMiddleware(apiRoute))
	apiRoute.HandleFunc("/info", getInfo).Methods("GET")
	apiRoute.HandleFunc("/config", getConfig).Methods("GET")
	apiRoute.HandleFunc("/config", putConfig).Methods("PUT")
	apiRoute.HandleFunc("/raw-config", getRawConfig).Methods("GET")
	apiRoute.HandleFunc("/raw-config", putRawConfig).Methods("PUT")
	apiRoute.HandleFunc("/lives", getAllLives).Methods("GET")
	apiRoute.HandleFunc("/lives", addLives).Methods("POST")
	apiRoute.HandleFunc("/lives/{id}", getLive).Methods("GET")
	apiRoute.HandleFunc("/lives/{id}", removeLive).Methods("DELETE")
	apiRoute.HandleFunc("/lives/{id}/{action}", mainHandler).Methods("GET")
	apiRoute.HandleFunc("/file/{path:.*}", getFileInfo).Methods("GET")
	apiRoute.HandleFunc("/lives/{id}/rtmp", addRtmp).Methods("POST")
	apiRoute.HandleFunc("/lives/{id}/{resource}/{action}", mainHandler).Methods("GET")
	apiRoute.Handle("/metrics", promhttp.Handler()) // 用于处理 Prometheus 监控数据

	// 设置静态文件服务
	m.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(instance.GetInstance(ctx).Config.OutPutPath))))

	// 设置 Web 应用程序
	fs, err := webapp.FS()
	if err != nil {
		instance.GetInstance(ctx).Logger.Fatal(err)
	}
	m.PathPrefix("/").Handler(http.FileServer(fs))

	// 启用 pprof 性能分析
	if instance.GetInstance(ctx).Config.Debug {
		m.PathPrefix("/debug/").Handler(http.DefaultServeMux)
	}
	return m
}

// NewServer 函数创建一个新的服务器实例。
func NewServer(ctx context.Context) *Server {
	inst := instance.GetInstance(ctx)
	config := inst.Config
	httpServer := &http.Server{
		Addr:    config.RPC.Bind,
		Handler: initMux(ctx),
	}
	server := &Server{server: httpServer}
	inst.Server = server
	return server
}

// Start 方法启动服务器。
func (s *Server) Start(ctx context.Context) error {
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Add(1)
	go func() {
		switch err := s.server.ListenAndServe(); err {
		case nil, http.ErrServerClosed:
		default:
			inst.Logger.Error(err)
		}
	}()
	inst.Logger.Infof("Server start at %s", s.server.Addr)
	return nil
}

// Close 方法关闭服务器。
func (s *Server) Close(ctx context.Context) {
	inst := instance.GetInstance(ctx)
	inst.WaitGroup.Done()
	ctx2, cancel := context.WithCancel(ctx)
	if err := s.server.Shutdown(ctx2); err != nil {
		inst.Logger.WithError(err).Error("failed to shutdown server")
	}
	defer cancel()
	inst.Logger.Infof("Server close")
}
