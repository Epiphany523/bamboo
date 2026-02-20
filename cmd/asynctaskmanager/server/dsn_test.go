package server

import (
	"testing"
	"time"
)

func TestMysqlDSN_Parse(t *testing.T) {
	tests := []struct {
		name     string
		dsn      MysqlDSN
		wantUser string
		wantPass string
		wantHost string
		wantPort int
		wantDB   string
	}{
		{
			name:     "完整 DSN",
			dsn:      "root:password@tcp(localhost:3306)/asynctask?charset=utf8mb4&parseTime=True&loc=Local",
			wantUser: "root",
			wantPass: "password",
			wantHost: "localhost",
			wantPort: 3306,
			wantDB:   "asynctask",
		},
		{
			name:     "不同端口",
			dsn:      "admin:secret123@tcp(192.168.1.100:3307)/mydb?charset=utf8mb4",
			wantUser: "admin",
			wantPass: "secret123",
			wantHost: "192.168.1.100",
			wantPort: 3307,
			wantDB:   "mydb",
		},
		{
			name:     "无密码",
			dsn:      "root:@tcp(localhost:3306)/testdb",
			wantUser: "root",
			wantPass: "",
			wantHost: "localhost",
			wantPort: 3306,
			wantDB:   "testdb",
		},
		{
			name:     "空 DSN 使用默认值",
			dsn:      "",
			wantUser: "root",
			wantPass: "",
			wantHost: "localhost",
			wantPort: 3306,
			wantDB:   "asynctask",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.dsn.Parse()

			if config.User != tt.wantUser {
				t.Errorf("User = %v, want %v", config.User, tt.wantUser)
			}
			if config.Password != tt.wantPass {
				t.Errorf("Password = %v, want %v", config.Password, tt.wantPass)
			}
			if config.Host != tt.wantHost {
				t.Errorf("Host = %v, want %v", config.Host, tt.wantHost)
			}
			if config.Port != tt.wantPort {
				t.Errorf("Port = %v, want %v", config.Port, tt.wantPort)
			}
			if config.Database != tt.wantDB {
				t.Errorf("Database = %v, want %v", config.Database, tt.wantDB)
			}

			// 验证连接池配置有默认值
			if config.MaxOpen != 100 {
				t.Errorf("MaxOpen = %v, want 100", config.MaxOpen)
			}
			if config.MaxIdle != 10 {
				t.Errorf("MaxIdle = %v, want 10", config.MaxIdle)
			}
			if config.MaxLife != time.Hour {
				t.Errorf("MaxLife = %v, want %v", config.MaxLife, time.Hour)
			}
		})
	}
}

func TestMysqlDSN_ParseEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		dsn  MysqlDSN
	}{
		{
			name: "复杂密码包含特殊字符",
			dsn:  "user:p@ssw0rd!@tcp(localhost:3306)/db",
		},
		{
			name: "域名主机",
			dsn:  "root:pass@tcp(db.example.com:3306)/production",
		},
		{
			name: "IPv6 地址",
			dsn:  "root:pass@tcp([::1]:3306)/testdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.dsn.Parse()
			
			// 至少应该有基本的配置
			if config.User == "" {
				t.Error("User should not be empty")
			}
			if config.Host == "" {
				t.Error("Host should not be empty")
			}
			if config.Database == "" {
				t.Error("Database should not be empty")
			}
		})
	}
}
