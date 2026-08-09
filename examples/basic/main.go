// basic 示例:缓存读写、TTL 过期与容量逐出。
package main

import (
	"fmt"
	"time"

	"github.com/lcylpzls/cachex"
)

// User 是示例缓存值。
type User struct {
	ID   int
	Name string
}

func main() {
	cache, err := cachex.New(
		cachex.WithCapacity(10000),
		cachex.WithDefaultTTL(5*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer cache.Close()

	cache.SetTTL("user:1", User{ID: 1, Name: "张三"}, 10*time.Minute)
	if v, ok := cache.Get("user:1"); ok {
		u := v.(User)
		fmt.Printf("命中用户:%d %s\n", u.ID, u.Name)
	}

	cache.Delete("user:1")
	if _, ok := cache.Get("user:1"); !ok {
		fmt.Println("删除后未命中")
	}
}
