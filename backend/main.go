package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var db *sql.DB // 表面db是sql.DB类型的指针。初始指向了nil

// RSS 结构体用于解析 XML
type Item struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
}

type Channel struct {
	Title string `xml:"title"`
	Items []Item `xml:"item"`
}

type RSS struct {
	Channel Channel `xml:"channel"`
}

type FeedResult struct {
	Author string
	Posts  []Item
}

func fetchRSS(url string, ch chan<- FeedResult, wg *sync.WaitGroup) {
	defer wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("请求失败:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取失败:", err)
		return
	}

	var rss RSS
	err = xml.Unmarshal(body, &rss)
	if err != nil {
		fmt.Println("XML解析失败:", err)
		return
	}

	ch <- FeedResult{Author: rss.Channel.Title, Posts: rss.Channel.Items}
}

func runDailyTask() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, now.Location())
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}

		timer := time.NewTimer(next.Sub(now))
		<-timer.C

		// 调用 trigger 的处理逻辑
		fmt.Println("开始执行每日定时任务 /trigger")
		req, _ := http.NewRequest("POST", "/trigger", nil)
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = req
		router := setupRouter()
		router.HandleContext(ctx)
	}
}

func basicAuthMiddleware(c *gin.Context) {
	user, pass, ok := c.Request.BasicAuth()
	if !ok || user != "admin" || pass != "KS3G7QPn" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"status":  "unauthorized",
			"message": "认证失败",
		})
		return
	}
	c.Next()
}
func setupRouter() *gin.Engine {

	var err error
	// 替换为你自己的连接信息
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://myuser:mypassword@localhost/mydatabase?sslmode=disable"
	}
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	// 自动创建 subscription 表（如果不存在）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS subscription (
			author TEXT PRIMARY KEY,
			value JSONB NOT NULL
		)
	`)
	if err != nil {
		panic(err)
	}

	// Disable Console Color
	// gin.DisableConsoleColor()

	r := gin.Default()

	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // 允许的前端域名
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"}, // 添加 Authorization
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true, // 如果需要携带 cookies，设为 true
		MaxAge:           12 * time.Hour,
	}

	r.Use(cors.New(config)) // 允许跨域请求

	// Ping test
	// r.GET("/ping", func(c *gin.Context) {
	// 	c.String(http.StatusOK, "pong")
	// })

	// Get author value
	r.GET("/api/v1/author/:name", func(c *gin.Context) {
		author := c.Params.ByName("name")
		var value string
		err := db.QueryRow("SELECT value FROM subscription WHERE author = $1", author).Scan(&value)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{"author": author, "status": "no value"})
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"author": author, "value": value})
		}
	})

	// Authorized group (uses gin.BasicAuth() middleware)
	// Same than:
	// authorized := r.Group("/")
	// authorized.Use(gin.BasicAuth(gin.Credentials{
	//	  "foo":  "bar",
	//	  "manu": "123",
	//}))
	// authorized := r.Group("/api/v1", gin.BasicAuth(gin.Accounts{
	// 	"admin": "KS3G7QPn",
	// }))
	authorized := r.Group("/api/v1")
	authorized.Use(basicAuthMiddleware)
	authorized.Use(cors.New(config)) // 保留原有的 CORS 设置

	/* example curl for /admin with basicauth header
	   Zm9vOmJhcg== is base64("foo:bar")

		curl -X POST \
	  	http://localhost:8080/admin \
	  	-H 'authorization: Basic Zm9vOmJhcg==' \
	  	-H 'content-type: application/json' \
	  	-d '{"value":"bar"}'
	*/
	authorized.POST("admin", func(c *gin.Context) {

		// Parse JSON
		var input struct {
			URL string `json:"url" binding:"required"`
		}

		var value struct {
			Author string          `json:"author" binding:"required"`
			URL    string          `json:"url" binding:"required"`
			Posts  json.RawMessage `json:"posts" binding:"required"`
		}

		if c.Bind(&input) == nil {
			// 去请求author和posts
			wg := &sync.WaitGroup{}

			ch := make(chan FeedResult, 1)

			wg.Add(1)
			go fetchRSS(input.URL, ch, wg)

			wg.Wait()
			close(ch)

			result := <-ch

			value.Author = result.Author
			value.URL = input.URL

			jsonRaw, err := json.Marshal(result.Posts)
			// fmt.Println("序列化结果:", string(jsonRaw))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			value.Posts = jsonRaw

			jsonData, err := json.Marshal(value)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON"})
				return
			}
			// UPSERT（插入或更新）
			_, err = db.Exec(`
				INSERT INTO subscription (author, value)
				VALUES ($1, $2)
				ON CONFLICT (author) DO UPDATE SET value = EXCLUDED.value
			`, value.Author, jsonData)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error": err.Error()})
			} else {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			}
		}
	})

	// POST 触发订阅
	authorized.POST("/trigger", func(c *gin.Context) {
		rows, err := db.Query("SELECT * FROM subscription")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var urls []string

		for rows.Next() {
			var author string
			var value []byte
			err := rows.Scan(&author, &value)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 解析 JSON 数据
			var data struct {
				URL   string `json:"url"`
				Posts []Item `json:"posts"`
			}

			err = json.Unmarshal(value, &data)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			urls = append(urls, data.URL)

		}

		wg := &sync.WaitGroup{}

		ch := make(chan FeedResult, len(urls))

		for _, url := range urls {
			wg.Add(1)
			go fetchRSS(url, ch, wg)
		}

		wg.Wait()
		close(ch)

		for result := range ch {
			fmt.Println("保存博客: ", result.Author, len(result.Posts))
			// 更新 subscription 表 value 字段，把Posts放到value的posts字段

			var jsonRaw json.RawMessage
			jsonRaw, err = json.Marshal(result.Posts)
			// fmt.Println("序列化结果:", string(jsonRaw))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			_, err = db.Exec(`
				UPDATE subscription SET value = jsonb_set(value, '{posts}', $1::jsonb, true)
				WHERE author = $2
			`, jsonRaw, result.Author)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		fmt.Println("所有RSS处理完毕。")
		c.JSON(http.StatusOK, gin.H{"status": "ok"})

	})

	// 首页接口，返回近一月更新的文章
	r.GET("/api/v1/index", func(c *gin.Context) {
		var data []Item
		rows, err := db.Query(`SELECT
			s.author,
			post->>'Title' AS title,
			post->>'Link' AS link,
			post->>'PubDate' AS pubdate
			FROM
			"subscription" s,
			LATERAL jsonb_array_elements(s.value->'posts') AS post
			WHERE
			(
				to_timestamp(post->>'PubDate', 'Dy, DD Mon YYYY HH24:MI:SS GMT')
				>= now() - interval '30 days'
			)`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		defer rows.Close()

		for rows.Next() {
			var author string
			var title string
			var link string
			var pubdate string
			err := rows.Scan(&author, &title, &link, &pubdate)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			data = append(data, Item{
				Title:   title,
				Link:    link,
				PubDate: pubdate,
			})
		}

		c.JSON(http.StatusOK, gin.H{"data": data})
	})

	// 登录接口
	r.POST("/api/v1/login", func(c *gin.Context) {
		var input struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if c.Bind(&input) == nil {
			// 验证用户名和密码
			if input.Username == "admin" && input.Password == "KS3G7QPn" {
				c.JSON(http.StatusOK, gin.H{"status": "ok", "token": "YWRtaW46S1MzRzdRUG4="})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			}
		}
	})

	// 正确地托管所有静态资源，并处理前端路由
	r.GET("/", func(c *gin.Context) {
		c.File("./dist/index.html")
	})
	r.Static("/static", "./dist")
	r.NoRoute(func(c *gin.Context) {
		c.File("./dist/index.html")
	})

	return r
}

func main() {
	r := setupRouter()
	// 启动定时任务
	go runDailyTask()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")
}
