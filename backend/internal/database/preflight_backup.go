package database

import (
	"context"
	"dockerun/internal/config"
	"fmt"
	mysql "github.com/go-sql-driver/mysql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func backupLegacyMySQL(cfg *config.Config) error {
	c, err := mysql.ParseDSN(cfg.MySQLDSN)
	if err != nil {
		return err
	}
	dir := filepath.Join(cfg.DataDir, "backups")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, fmt.Sprintf("pre-v1-%d.sql", time.Now().UnixNano()))
	f, err := os.OpenFile(path+".partial", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	args := []string{"--single-transaction", "--user=" + c.User}
	if c.Net == "unix" {
		args = append(args, "--socket="+c.Addr)
	} else {
		host, port, _ := strings.Cut(c.Addr, ":")
		args = append(args, "--host="+host)
		if port != "" {
			args = append(args, "--port="+port)
		}
	}
	args = append(args, c.DBName)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+c.Passwd)
	cmd.Stdout = f
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("升级前 MySQL 备份失败，未执行迁移：%w", err)
	}
	if err = f.Sync(); err != nil {
		return err
	}
	return os.Rename(path+".partial", path)
}
