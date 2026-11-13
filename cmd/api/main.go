package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/JoyElliot/zwOA/internal/api"
	"github.com/JoyElliot/zwOA/internal/modules/chat"
	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/JoyElliot/zwOA/internal/platform/httpserver"
	"github.com/JoyElliot/zwOA/internal/platform/sqlite"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		log.Printf("API 服务退出：%v", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	path := strings.TrimSpace(os.Getenv("ZWOA_DB_PATH"))
	if path == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return fmt.Errorf("无法定位数据目录，请设置 ZWOA_DB_PATH：%w", err)
		}
		path = filepath.Join(base, "zwOA", "zwoa.db")
	}
	if len(os.Args) > 1 && (len(os.Args) != 4 || os.Args[1] != "init-admin") {
		return errors.New(`用法：go run ./cmd/api init-admin <登录名> "<姓名>"`)
	}
	db, err := sqlite.Open(ctx, path)
	if err != nil {
		return fmt.Errorf("打开数据库：%w", err)
	}
	defer db.Close()
	accounts := organization.New(db)
	if len(os.Args) > 1 {
		return initializeAdmin(ctx, accounts, os.Args[2], os.Args[3])
	}
	server := api.New(httpserver.ConfigFromEnv(), accounts, chat.New(db))
	if err := server.Start(); err != nil {
		return err
	}
	<-ctx.Done()
	return server.Shutdown()
}

func initializeAdmin(ctx context.Context, store *organization.Store, username, name string) error {
	password, supplied := os.LookupEnv("ZWOA_INIT_ADMIN_PASSWORD")
	if !supplied {
		fd := int(os.Stdin.Fd())
		fmt.Fprint(os.Stderr, "输入超级管理员密码（至少 12 个字符）：")
		first, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("无法读取密码：%w", err)
		}
		fmt.Fprint(os.Stderr, "再次输入密码：")
		second, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("无法读取密码：%w", err)
		}
		if string(first) != string(second) {
			return errors.New("两次输入的密码不一致")
		}
		password = string(first)
	}
	if err := store.Bootstrap(ctx, username, name, password); err != nil {
		if errors.Is(err, organization.ErrNotFound) {
			return errors.New("数据库已经初始化")
		}
		if errors.Is(err, organization.ErrInvalid) {
			return errors.New("登录名需为 3–40 位字母、数字或 . _ -，以字母或数字开头，姓名不能为空，密码至少 12 个字符且不超过 256 字节")
		}
		return err
	}
	fmt.Fprintln(os.Stderr, "超级管理员已创建。")
	return nil
}
