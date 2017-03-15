//---------------------------------------------
package main

//---------------------------------------------
import (
	FMT "fmt"
	OS "os"
	SORT "sort"
	STRCONV "strconv"
	SYNC "sync"

	LOG "github.com/Sirupsen/logrus"
	CLI "gopkg.in/urfave/cli.v2"

	COMMON "FKGoServer/FKTools_CuiClient/Common"
	SARAMA "github.com/Shopify/sarama"
	GOCUI "github.com/jroimartin/gocui"
)

//---------------------------------------------
const (
	cmdPlay = iota
	cmdBack
	cmdForward
	cmdJumpOldest
	cmdJumpNewest
)

//---------------------------------------------
var (
	active    = 0
	viewNames = []string{"topic", "data"}
	control   = make(chan int, 1)
	client    SARAMA.Client
	topic     string
	partition int
	brokers   []string
	wg        SYNC.WaitGroup
	stop      = make(chan struct{})
	paused    bool
)

//---------------------------------------------
func main() {
	app := &CLI.App{
		Name:    "FKCuiClient",
		Usage:   `a kafka topic player`,
		Version: "1.0",
		Flags: []CLI.Flag{
			&CLI.StringSliceFlag{
				Name:  "brokers, b",
				Value: CLI.NewStringSlice("localhost:9092"),
				Usage: "kafka brokers address",
			},
			&CLI.StringFlag{
				Name:  "log",
				Value: "FKCuiClient.log",
				Usage: "log file",
			},
		},
		Action: processor,
	}
	app.Run(OS.Args)
}

//---------------------------------------------
func processor(c *CLI.Context) error {
	brokers = c.StringSlice("brokers")
	out := c.String("log")
	if out != "" {
		if f, err := OS.OpenFile(out, OS.O_RDWR|OS.O_CREATE, 0666); err == nil {
			LOG.SetOutput(f)
		} else {
			panic(err)
		}
	}

	var err error
	client, err = SARAMA.NewClient(brokers, nil)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	g, err := GOCUI.NewGui(GOCUI.OutputNormal)
	if err != nil {
		LOG.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)
	g.Highlight = true
	g.SelFgColor = GOCUI.ColorMagenta

	if err := g.SetKeybinding("", GOCUI.KeyCtrlC, GOCUI.ModNone, quit); err != nil {
		LOG.Panicln(err)
	}
	if err := g.SetKeybinding("", GOCUI.KeyTab, GOCUI.ModNone, nextView); err != nil {
		LOG.Panicln(err)
	}
	if err := g.SetKeybinding("topic", GOCUI.KeyArrowDown, GOCUI.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("topic", GOCUI.KeyArrowUp, GOCUI.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("topic", GOCUI.KeyEnter, GOCUI.ModNone, selectTopic); err != nil {
		return err
	}
	if err := g.SetKeybinding("partition", GOCUI.KeyEnter, GOCUI.ModNone, selectPartition); err != nil {
		return err
	}
	if err := g.SetKeybinding("partition", GOCUI.KeyArrowDown, GOCUI.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("partition", GOCUI.KeyArrowUp, GOCUI.ModNone, cursorUp); err != nil {
		return err
	}

	if err := g.SetKeybinding("data", GOCUI.KeySpace, GOCUI.ModNone,
		func(g *GOCUI.Gui, v *GOCUI.View) error {
			select {
			case control <- cmdPlay:
			default:
			}
			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("data", GOCUI.KeyArrowLeft, GOCUI.ModNone,
		func(g *GOCUI.Gui, v *GOCUI.View) error {
			select {
			case control <- cmdBack:
			default:
			}
			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("data", GOCUI.KeyArrowRight, GOCUI.ModNone,
		func(g *GOCUI.Gui, v *GOCUI.View) error {
			select {
			case control <- cmdForward:
			default:
			}
			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("data", '[', GOCUI.ModNone,
		func(g *GOCUI.Gui, v *GOCUI.View) error {
			select {
			case control <- cmdJumpOldest:
			default:
			}
			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("data", ']', GOCUI.ModNone,
		func(g *GOCUI.Gui, v *GOCUI.View) error {
			select {
			case control <- cmdJumpNewest:
			default:
			}
			return nil
		}); err != nil {
		return err
	}

	if err := g.MainLoop(); err != nil && err != GOCUI.ErrQuit {
		LOG.Panicln(err)
	}
	return nil
}

//---------------------------------------------
// Model
func player(g *GOCUI.Gui, topic string, partition int32, offset int64) {
	wg.Add(1)
	defer wg.Done()

	consumer, err := SARAMA.NewConsumerFromClient(client)
	if err != nil {
		panic(err)
	}
	defer consumer.Close()
	partitionConsumer, err := consumer.ConsumePartition(topic, int32(partition), int64(offset))
	if err != nil {
		panic(err)
	}

	defer func() {
		partitionConsumer.Close()
	}()

	chMessage := partitionConsumer.Messages()
	for {
		select {
		case msg := <-chMessage:
			offset = msg.Offset
			g.Execute(func(g *GOCUI.Gui) error {
				v, _ := g.View("data")
				v.Clear()
				v.Title = FMT.Sprintf("OFFSET:%v KEY:%v TIMESTAMP:%v", msg.Offset, string(msg.Key), msg.Timestamp)
				FMT.Fprintln(v, string(msg.Value))
				return nil
			})
			if paused {
				chMessage = nil
			}
		case c := <-control:
			switch c {
			case cmdPlay:
				paused = !paused
				if paused {
					chMessage = nil
				} else {
					chMessage = partitionConsumer.Messages()
				}
			case cmdBack:
				if offset-1 < 0 {
					continue
				}
				paused = true
				partitionConsumer.Close()
				partitionConsumer, err = consumer.ConsumePartition(topic, int32(partition), int64(offset-1))
				if err != nil {
					panic(err)
				}
				chMessage = partitionConsumer.Messages()
			case cmdForward:
				paused = true
				partitionConsumer.Close()
				partitionConsumer, err = consumer.ConsumePartition(topic, int32(partition), int64(offset+1))
				if err != nil {
					partitionConsumer, err = consumer.ConsumePartition(topic, int32(partition), SARAMA.OffsetNewest)
					if err != nil {
						panic(err)
					}
				}
				chMessage = partitionConsumer.Messages()
			case cmdJumpOldest:
				paused = true
				partitionConsumer.Close()
				partitionConsumer, err = consumer.ConsumePartition(topic, int32(partition), SARAMA.OffsetOldest)
				if err != nil {
					panic(err)
				}
				chMessage = partitionConsumer.Messages()
			case cmdJumpNewest:
				paused = false
				partitionConsumer.Close()
				partitionConsumer, err = consumer.ConsumePartition(topic, int32(partition), SARAMA.OffsetNewest)
				if err != nil {
					panic(err)
				}
				chMessage = partitionConsumer.Messages()
			}
			refreshInfo(g)
		case <-stop:
			return
		}
	}
}

// View
func nextView(g *GOCUI.Gui, v *GOCUI.View) error {
	active = (active + 1) % len(viewNames)
	g.SetCurrentView(viewNames[active])
	return nil
}

func cursorUp(g *GOCUI.Gui, v *GOCUI.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorDown(g *GOCUI.Gui, v *GOCUI.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if l, err := v.Line(cy + 1); err == nil {
			if l == "" {
				return nil
			}
		} else {
			return err
		}

		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func layout(g *GOCUI.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("topic", 0, 0, 10, maxY-1); err != nil {
		if err != GOCUI.ErrUnknownView {
			return err
		}
		v.Title = "TOPIC"
		topics, err := client.Topics()
		if err != nil {
			panic(err)
		}

		SORT.Sort(COMMON.NewWrapperWith(topics, func(i, j int) bool {
			if topics[i] < topics[j] {
				return true
			}
			return false
		}))

		for k := range topics {
			FMT.Fprintln(v, topics[k])
		}
		v.Highlight = true
		v.SelBgColor = GOCUI.ColorGreen
		v.SelFgColor = GOCUI.ColorBlack
		g.SetCurrentView("topic")
	}

	if v, err := g.SetView("play", 11, maxY-10, 11+30, maxY-1); err != nil {
		if err != GOCUI.ErrUnknownView {
			return err
		}
		v.Title = "PLAY"
		FMT.Fprintln(v, "Space: Pause/Play")
		FMT.Fprintln(v, "← → : Back/Forward")
		FMT.Fprintln(v, "[: Jump to oldest")
		FMT.Fprintln(v, "]: Jump to newest")
	}

	if v, err := g.SetView("info", 11+30+1, maxY-10, 11+30+50, maxY-1); err != nil {
		if err != GOCUI.ErrUnknownView {
			return err
		}
		v.Title = "INFO"
	}

	if v, err := g.SetView("options", 11+30+50+1, maxY-10, maxX-1, maxY-1); err != nil {
		if err != GOCUI.ErrUnknownView {
			return err
		}
		v.Title = "GLOBAL OPTIONS"
		FMT.Fprintln(v, "Tab: Next View")
		FMT.Fprintln(v, "^C: Exit")
	}

	if v, err := g.SetView("data", 11, 0, maxX-1, maxY-11); err != nil {
		if err != GOCUI.ErrUnknownView {
			return err
		}
		v.Title = "DATA"
		v.Wrap = true
	}

	return nil
}

// Control
func selectTopic(g *GOCUI.Gui, v *GOCUI.View) error {
	var l string
	var err error

	_, cy := v.Cursor()
	if l, err = v.Line(cy); err != nil {
		l = ""
	}
	parts, err := client.Partitions(l)
	if err != nil {
		panic(err)
	}

	topic = l
	maxX, maxY := g.Size()
	if v, err := g.SetView("partition", maxX/2-10, maxY/2, maxX/2+10, maxY/2+2); err != nil {
		if err != GOCUI.ErrUnknownView {
			return err
		}
		for k := range parts {
			FMT.Fprintln(v, parts[k])
		}
		if _, err := g.SetCurrentView("partition"); err != nil {
			return err
		}
		v.Title = "Select Partition"
	}
	return nil
}

func selectPartition(g *GOCUI.Gui, v *GOCUI.View) error {
	// get partition
	var l string
	var err error

	_, cy := v.Cursor()
	if l, err = v.Line(cy); err != nil {
		l = ""
	}
	if err := g.DeleteView("partition"); err != nil {
		return err
	}
	if _, err := g.SetCurrentView("data"); err != nil {
		return err
	}
	dataView, _ := g.View("data")
	dataView.Clear()
	active = 1

	// turn off player
	close(stop)
	stop = make(chan struct{})

	// wait until player exists
	wg.Wait()

	// create new player for partition
	partition, _ = STRCONV.Atoi(l)
	go player(g, topic, int32(partition), SARAMA.OffsetNewest)

	// update info
	refreshInfo(g)
	return nil
}

func refreshInfo(g *GOCUI.Gui) {
	g.Execute(func(g *GOCUI.Gui) error {
		infoview, _ := g.View("info")
		infoview.Clear()
		FMT.Fprintf(infoview, "Brokers: %v\n", brokers)
		FMT.Fprintf(infoview, "Topic: %v\n", topic)
		FMT.Fprintf(infoview, "Partition: %v\n", partition)
		replicas, _ := client.Replicas(topic, int32(partition))
		FMT.Fprintf(infoview, "Replicas: %v\n", replicas)
		broker, _ := client.Leader(topic, int32(partition))
		FMT.Fprintf(infoview, "Leader: %v\n", broker.ID())
		if paused {
			FMT.Fprintf(infoview, "\033[31mPAUSED\033[0m\n")
		} else {
			FMT.Fprintf(infoview, "\033[32mPLAYING\033[0m\n")
		}
		return nil
	})
}

func quit(g *GOCUI.Gui, v *GOCUI.View) error {
	return GOCUI.ErrQuit
}
