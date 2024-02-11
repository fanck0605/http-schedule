package main

import (
	"context"
	"errors"
	"github.com/fanck0605/http-schedule/schedule"
	"github.com/fanck0605/http-schedule/utils"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
func copyBuffer(dst gin.ResponseWriter, src io.Reader, buf []byte) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = io.ErrShortWrite
				}
			}
			dst.Flush() // FIXME 为 CopyBuffer 加上这个 Flush？
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func main() {
	scheduler := schedule.NewScheduler(1024)
	scheduler.Start()

	client := &http.Client{}

	router := gin.Default()
	router.Any("*uri", func(ctx *gin.Context) {
		log.Printf("Receive task %s", ctx.Request.RequestURI)
		taskCtx, taskDone := context.WithCancel(context.Background())
		defer taskDone()

		ready := make(chan struct{})
		scheduler.Push(&schedule.Task{
			Priority: utils.GetPriority(ctx.Request),
			Weight:   utils.GetWeight(ctx.Request),
			Context:  taskCtx,
			Ready:    ready,
		})
		<-ready

		log.Printf("Task ready to run %s", ctx.Request.RequestURI)
		url := utils.GetRequestURL(ctx.Request)
		if proxyReq, reqErr := http.NewRequest(ctx.Request.Method, url, ctx.Request.Body); reqErr != nil {
			_ = ctx.AbortWithError(500, reqErr)
		} else {
			proxyReq.Header = make(http.Header)
			for h, val := range ctx.Request.Header {
				proxyReq.Header[h] = val
			}
			if resp, err := client.Do(proxyReq); err != nil {
				_ = ctx.AbortWithError(500, err)
			} else {
				// 回收资源
				defer func() {
					if err := resp.Body.Close(); err != nil {
						log.Println(err)
					}
				}()
				ctx.Writer.WriteHeader(resp.StatusCode)
				headerWriter := ctx.Writer.Header()
				for name, values := range resp.Header {
					for _, value := range values {
						headerWriter.Add(name, value)
					}
				}
				ctx.Writer.WriteHeaderNow()

				// TODO zero copy
				if _, err := copyBuffer(ctx.Writer, resp.Body, make([]byte, 4096)); err != nil {
					log.Println(err)
				}
			}
			log.Printf("task taskDone %s", ctx.Request.RequestURI)
		}
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		// service connections
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
	// The Task is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %s\n", err)
	}
	scheduler.Stop()
	log.Println("Server exiting")
}
