package configs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewConfig 测试NewConfig函数。
func TestNewConfig(t *testing.T) {
	// 指定配置文件路径
	configFilePath := "../../config.yml"

	// 创建新的配置对象
	c, err := NewConfigWithFile(configFilePath)

	// 断言测试结果
	assert.NoError(t, err)
	assert.Equal(t, configFilePath, c.File)
}

// TestRPC_Verify 测试RPC对象的验证函数。
func TestRPC_Verify(t *testing.T) {
	// 创建一个RPC对象
	rpc := &RPC{}

	// 验证RPC对象，预期不会出错
	assert.NoError(t, rpc.verify())

	// 设置RPC的Bind字段
	rpc.Bind = "foo@bar"

	// 再次验证RPC对象，预期不会出错
	assert.NoError(t, rpc.verify())

	// 启用RPC
	rpc.Enable = true

	// 验证RPC对象，预期会出错
	assert.Error(t, rpc.verify())
}

// TestConfig_Verify 测试Config对象的验证函数。
func TestConfig_Verify(t *testing.T) {
	// 创建一个Config对象
	cfg := &Config{}

	// 验证Config对象，预期会出错
	assert.Error(t, cfg.Verify())

	// 设置Config对象的初始值
	cfg = &Config{
		RPC:        defaultRPC,
		Interval:   30,
		OutPutPath: os.TempDir(),
	}

	// 验证Config对象，预期不会出错
	assert.NoError(t, cfg.Verify())

	// 修改Interval为0，预期会出错
	cfg.Interval = 0
	assert.Error(t, cfg.Verify())

	// 恢复Interval的值，修改OutPutPath为无效路径，预期会出错
	cfg.Interval = 30
	cfg.OutPutPath = "foobar"
	assert.Error(t, cfg.Verify())

	// 恢复OutPutPath的值，将RPC的Enable字段设置为false，预期会出错
	cfg.OutPutPath = os.TempDir()
	cfg.RPC.Enable = false
	assert.Error(t, cfg.Verify())
}
