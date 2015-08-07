package golongapprunner

import (
	"log"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParsingCmdLine(t *testing.T) {
	Convey("Given cmd line", t, func() {
		Convey("When it has quotes", func() {
			Convey("It should work well", func() {
				args := StringToArgs(`ping ya.ru  -t "hell\"o w\\orld"  "hello world"`)
				for _, v := range args {
					log.Println("-", "["+v+"]")
				}
				So(len(args), ShouldEqual, 5)
			})
		})
	})
}

func TestPing(t *testing.T) {
	//t.Skip()
	return
	Convey("Given it should not crash", t, func() {
		Convey("When running test ping app", func() {

			Convey("It (simple) should stop itself", func() {
				r, ch := RunSimple("ping ya.ru")

				// channel is closed when both stderr and stdout are closed
				for x := range LogsChan_toStringChan(ch, "> ", "! ") {
					println(x)
				}

				log.Print(r.Summary())

				So(r.StoppedItself, ShouldBeTrue)
				So(r.TimeoutHappened, ShouldBeFalse)
				So(r.ForcefullyKilled, ShouldBeFalse)
			})

			//Convey("It (simple ffmpeg) should stop time out", func() {
			//	r, ch := PrepareSimpleRun("c:\\ffmpeg -i rtmp://server/live/stream -acodec copy -vcodec copy -f flv rtmp://server/live/stream2")
			//	r.Timeout = time.Second * 15
			//	//r.SplitFunc
			//	go r.Start()

			//	// channel is closed when both stderr and stdout are closed
			//	for x := range ByteDataChan_toStringChan(ch, "> ", "! ") {
			//		println(x)
			//	}

			//	log.Print(r.Summary())

			//	So(r.StoppedItself, ShouldBeFalse)
			//	So(r.TimeoutHappened, ShouldBeTrue)
			//	So(r.ForcefullyKilled, ShouldBeFalse)
			//})

			//			Convey("It (simple) should work well with timeouts", func() {
			//				r, ch := RunLimitedInTime("ping ya.ru -t", time.Second)

			//				// channel is closed when both stderr and stdout are closed
			//				for x := range ch {
			//					log.Println(ternString(x.stderr, "e-", "o-") + string(x.data))
			//				}

			//				log.Print(r.Summary())

			//				So(r.StoppedItself, ShouldBeFalse)
			//				So(r.TimeoutHappened, ShouldBeTrue)
			//				So(r.ForcefullyKilled, ShouldBeFalse)
			//			})
		})
	})
}
