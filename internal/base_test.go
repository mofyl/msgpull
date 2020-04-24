package internal

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"me.xiaoka.bee/internal/natsop"
)

const (
	Test = "test"
)

func notifySignal(wg *sync.WaitGroup, f func()) {
	go func() {

		wg.Add(1)
		c := make(chan os.Signal)

		signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

		<-c
		f()
		wg.Done()
	}()

}

func TestReq(t *testing.T) {

	// notifyChan := make(chan *msg.MsgInfo)
	natsop.Connect(nil)

	msg, err := natsop.Call(Test, 0, []byte("hello world"), 3*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Req Come msg ", msg)
	// msg, err = natsop.Call(msg.ReplyTopic, msg.ReqMsgId, []byte("hello world"), 3*time.Second)
	ticker := time.NewTicker(1 * time.Second)

	// time.Sleep(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			ticker.Stop()
			natsop.Close()
			return
		default:

			msg, err = natsop.Call(msg.ReplyTopic, msg.MsgId, []byte("hello world"), 3*time.Second)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}

}

func TestReply(t *testing.T) {
	// sum := 0
	natsop.Connect(nil)
	c := make(chan *natsop.MsgInfo)
	natsop.SubChan(Test, c)

	msg := <-c
	var err error
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			ticker.Stop()

			natsop.Close()
			return
		default:

			msg, err = natsop.Call(msg.ReplyTopic, msg.MsgId, []byte("111111"), 3*time.Second)
			// natsop.Reply(m.ReplyTopic, m.ReqMsgId, []byte("111111"), m.Timeout)
			// msg, err := natsop.Call(m.ReplyTopic, m.ReqMsgId, []byte("111111"), 3*time.Second)

			if err != nil {
				fmt.Println("err is ", err)
				return
			}

			fmt.Println("Come Reply ", msg)
			// sum++

		}
	}

}

func BenchmarkReq(b *testing.B) {
	// notifyChan := make(chan *msg.MsgInfo)
	var err error
	var msg *natsop.MsgInfo
	// time.Sleep(1 * time.Second)
	natsop.Connect(nil)
	// defer natsop.Close()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		msg, err = natsop.Call(Test, 0, []byte("hello world"), 3*time.Second)

		if err != nil {
			fmt.Println(err)
			continue
			// return
		}
		if msg != nil {
			fmt.Println(string(msg.Data))
		}
	}

	// natsop.Close()

}

func BenchmarkReqReply(b *testing.B) {

	wg := &sync.WaitGroup{}
	natsop.Connect(nil)
	fmt.Println("BenchmarkReqReply")
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan *natsop.MsgInfo)
	natsop.SubChan(Test, c)

	go func() {
		http.ListenAndServe(":8080", nil)
	}()
	notifySignal(wg, func() {
		cancel()
		natsop.Close()
		return
	})

	wg.Add(1)
	go func() {
		sum := 0
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Reply sum ", sum)
				wg.Done()
				return
			case m := <-c:

				fmt.Println("Come Test ", string(m.Data))

				// natsop.Reply(m.ReplyTopic, m.ReqMsgId, []byte("111111"), m.Timeout)
				err := natsop.Send(m.ReplyTopic, m.MsgId, m.Timeout, []byte("1111111"))
				if err != nil {
					fmt.Println(err)
					continue
				}
				sum++
				fmt.Println("Reply sum ", sum)
			}
		}

	}()
	// time.Sleep(1 * time.Second)
	b.ResetTimer()
	sum := 0
	for i := 0; i < b.N; i++ {
		msg, err := natsop.Call(Test, 0, []byte("hello world"), 3*time.Second)

		if err != nil {
			fmt.Println(err)
			// return
			continue
		}
		// fmt.Println(msg)
		if msg != nil {
			sum++
			fmt.Println("cal sum ", sum, string(msg.Data))
		} else {
			fmt.Println("msg is nil")
		}

	}
	wg.Wait()
	// natsop.Close()
}

func TestCallReply(t *testing.T) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	natsop.Connect(nil)
	notifySignal(wg, func() {
		cancel()
		natsop.Close()
		return
	})

	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	c := make(chan *natsop.MsgInfo)
	natsop.SubChan(Test, c)
	wg.Add(1)
	go func() {

		sum := 0
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Reply sum ", sum)
				wg.Done()
				return
			case m := <-c:

				fmt.Println("Come Test ", m)

				// natsop.Reply(m.ReplyTopic, m.ReqMsgId, []byte("111111"), m.Timeout)
				natsop.Call(m.ReplyTopic, m.ReqMsgId, []byte("111111"), 3*time.Second)

				sum++
				fmt.Println("Reply sum ", sum)
			}
		}

	}()

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		sum := 0
		for {
			select {
			case <-ticker.C:
				fmt.Println("end.... ")
				return
			default:
				msg, err := natsop.Call(Test, 0, []byte("hello world"), 3*time.Second)

				if err != nil {
					fmt.Println(err)
					// return
					continue
				}
				// fmt.Println(msg)
				sum++
				if msg != nil {

					fmt.Println("cal sum ", sum, string(msg.Data))
				} else {
					fmt.Println("msg is nil")
				}
			}
		}
	}()
	wg.Wait()

}

func TestReqReply(t *testing.T) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	natsop.Connect(nil)
	notifySignal(wg, func() {
		cancel()
		natsop.Close()
		return
	})

	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	c := make(chan *natsop.MsgInfo)
	natsop.SubChan(Test, c)
	wg.Add(1)
	go func() {

		sum := 0
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Reply sum ", sum)
				wg.Done()
				return
			case m := <-c:

				fmt.Println("Come Test ", m)

				// natsop.Reply(m.ReplyTopic, m.ReqMsgId, []byte("111111"), m.Timeout)
				natsop.Send(m.ReplyTopic, m.ReqMsgId, m.Timeout, []byte("111111"))

				sum++
				fmt.Println("Reply sum ", sum)
			}
		}

	}()

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		sum := 0
		for {
			select {
			case <-ticker.C:
				fmt.Println("end.... ")
				return
			default:
				msg, err := natsop.Call(Test, 0, []byte("hello world"), 3*time.Second)

				if err != nil {
					fmt.Println(err)
					// return
					continue
				}
				// fmt.Println(msg)
				sum++
				if msg != nil {

					fmt.Println("cal sum ", sum, string(msg.Data))
				} else {
					fmt.Println("msg is nil")
				}
			}
		}
	}()
	wg.Wait()

}
func TestSub(t *testing.T) {
	wg := &sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	sum := 0

	natsop.Connect(nil)
	c := make(chan *natsop.MsgInfo)
	sub, err := natsop.SubChan(Test, c)
	notifySignal(wg, func() {
		cancel()
		natsop.Close()
		sub.Unsubscribe()
		fmt.Println(sum)
		return
	})

	if err != nil {
		fmt.Println("1231231")
		return
	}
	wg.Add(1)
	go func() {
		for {
			select {
			case m := <-c:
				fmt.Println(string(m.Data))
				sum++
			case <-ctx.Done():
				wg.Done()
				return
			}
		}
	}()
	wg.Wait()
	// nc, err := nats.Connect(nats.DefaultOptions.Url)

	// if err != nil {
	//   fmt.Println("qweqwe")
	//   return
	// }

	// c := make(chan *nats.Msg, 1024)
	// _, err = nc.ChanQueueSubscribe(Test, Test, c)

	// // err := natsop.SubChan(Test, c)

	// if err != nil {
	//   fmt.Println("123123123")
	//   return
	// }

	// for {
	//   select {
	//   case m := <-c:
	//     fmt.Println(m.Data)
	//   }
	// }

}

func TestPub(t *testing.T) {

	natsop.Connect(nil)

	// natsop.Send(Test, "", 0, []byte("Hello World"))
	// time.Sleep(1 * time.Second)
	// nc, err := nats.Connect(nats.DefaultOptions.Url)

	// if err != nil {
	//   fmt.Println("qweqwe")
	//   return
	// }

	// err = nc.Publish(Test, []byte("Hello World"))
	// // nc.Flush()

	// if err != nil {
	//   fmt.Println("public fail")
	// }
	// fmt.Println("Success")
	// time.Sleep(1 * time.Second)
	// natsop.Close()
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			natsop.Close()
			return
		default:
			natsop.Send(Test, 0, 0, []byte("Hello World"))
		}
	}

}
