package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"dockerun/internal/builtin"
	"dockerun/internal/config"
	"dockerun/internal/database"
	"dockerun/internal/release"
	"encoding/json"
	"fmt"
	mysql "github.com/go-sql-driver/mysql"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func Create(ctx context.Context) (string, error) {
	for !release.ExecutionGate.TryLock() {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	defer release.ExecutionGate.Unlock()
	ro := builtin.ReadOnly()
	builtin.SetReadOnly(true)
	defer builtin.SetReadOnly(ro)
	cfg := config.Load()
	dir, err := os.MkdirTemp(filepath.Join(cfg.DataDir, "backups"), "snapshot-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	dbFile := "metadata.db"
	if cfg.DatabaseType == "mysql" {
		dbFile = "metadata.sql"
		err = dumpMySQL(ctx, cfg.MySQLDSN, filepath.Join(dir, dbFile))
	} else {
		err = database.DB.WithContext(ctx).Exec("VACUUM INTO ?", filepath.Join(dir, dbFile)).Error
	}
	if err != nil {
		return "", err
	}
	path := filepath.Join(cfg.DataDir, "backups", time.Now().UTC().Format("20060102T150405.000000000Z")+".tar.gz")
	f, err := os.OpenFile(path+".partial", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	write := func(name, path string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("备份不接受非普通文件: %s", path)
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		if err = tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: info.Size(), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		_, err = io.Copy(tw, contextReader{ctx: ctx, reader: file})
		return err
	}
	meta, _ := json.Marshal(map[string]any{"schema_version": 1, "complete": true, "database_type": cfg.DatabaseType, "database_file": dbFile, "created_at": time.Now().UTC(), "contents": []string{"metadata", "keys", "registry", "releases", "uploads", "jobs"}})
	os.WriteFile(filepath.Join(dir, "backup.json"), meta, 0600)
	for _, name := range []string{dbFile, "backup.json"} {
		if err = write(name, filepath.Join(dir, name)); err != nil {
			return "", err
		}
	}
	for _, root := range []string{"keys", "registry", "releases", "uploads", "jobs"} {
		err = filepath.WalkDir(filepath.Join(cfg.DataDir, root), func(path string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(cfg.DataDir, path)
			return write(filepath.ToSlash(rel), path)
		})
		if err != nil {
			return "", err
		}
	}
	if err = tw.Close(); err != nil {
		return "", err
	}
	if err = gz.Close(); err != nil {
		return "", err
	}
	if err = f.Sync(); err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(path+".partial", path); err != nil {
		return "", err
	}
	return path, nil
}
func dumpMySQL(ctx context.Context, dsn, path string) error {
	c, err := mysql.ParseDSN(dsn)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer out.Close()
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
	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+c.Passwd)
	cmd.Stdout = out
	return cmd.Run()
}
func Restore(file, destination string) error {
	if entries, err := os.ReadDir(destination); err == nil && len(entries) > 0 {
		return fmt.Errorf("恢复目标必须是新的空目录；请停止 Dockerun 后切换到恢复的数据目录")
	}
	if err := os.MkdirAll(destination, 0700); err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			return fmt.Errorf("备份含非普通文件")
		}
		p, err := release.SafePath(destination, h.Name)
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			return err
		}
		out, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		_, err = io.CopyN(out, tr, h.Size)
		out.Close()
		if err != nil {
			return err
		}
	}
	if _, err = io.Copy(io.Discard, gz); err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(destination, "backup.json"))
	if err != nil {
		return err
	}
	var meta struct {
		SchemaVersion int    `json:"schema_version"`
		Complete      bool   `json:"complete"`
		DatabaseType  string `json:"database_type"`
	}
	if err = json.Unmarshal(b, &meta); err != nil || meta.SchemaVersion != 1 || !meta.Complete {
		return fmt.Errorf("不是完整 Dockerun 备份")
	}
	if meta.DatabaseType == "mysql" {
		return fmt.Errorf("材料已恢复；请将 metadata.sql 恢复到专用 MySQL 数据库后启动 Dockerun")
	}
	return os.Rename(filepath.Join(destination, "metadata.db"), filepath.Join(destination, "dockerun.db"))
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(b)
}
