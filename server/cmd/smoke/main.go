// smoke：headless 端到端冒烟客户端——两个 WS 客户端接入同一场本地对局并打满。
// 用法：先起 judge 与 codewar 服务，再运行
//
//	go run ./cmd/smoke
//
// 客户端按当前回合题目提交对应正确解 → 双方秒解 → 提前结算；
// 五回合互扣 22 点后双双归零 → 验证互相击杀平局路径。全链路约 3 秒。
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"codewar/server/gen/matchpb"
	"codewar/server/internal/question"
	"codewar/server/internal/question/templates"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
)

// solutions：模板 ID → 参考解（04 号票：出题工厂实例随机化，但函数语义固定，
// 同一模板的参考解可解任意实例；问题 ID 形如 "<模板ID>@<种子>:<形式>"）。
var solutions = buildSolutions()

func buildSolutions() map[string]string {
	out := map[string]string{}
	for _, tm := range templates.All() {
		in, err := tm.Generate(question.NewRNG(question.DeriveSeed("smoke", tm.Meta().ID)))
		if err != nil {
			panic(err)
		}
		out[tm.Meta().ID] = in.AnswerKey
	}
	return out
}

// templateIDOf：从问题 ID 还原模板 ID。
func templateIDOf(problemID string) string {
	if i := strings.Index(problemID, "@"); i >= 0 {
		return problemID[:i]
	}
	return problemID
}

func main() {
	addr := flag.String("addr", "localhost:3001", "codewar 服务地址")
	flag.Parse()

	done := make(chan *matchpb.MatchEnded, 2)
	for _, p := range []string{"p1", "p2"} {
		go play(*addr, p, done)
	}
	a, b := <-done, <-done
	fmt.Printf("对局终局: winner=%q reason=%q\n", a.Winner, a.Reason)
	fmt.Printf("另一连接视图: winner=%q reason=%q\n", b.Winner, b.Reason)
	if a.Winner != b.Winner || a.Reason != b.Reason {
		log.Fatalf("两连接终局视图不一致")
	}
	if a.Reason != "ko" || a.Winner != "" {
		log.Fatalf("预期互相击杀平局，实际 %q/%q", a.Winner, a.Reason)
	}
	fmt.Println("SMOKE OK")
}

func play(addr, player string, done chan<- *matchpb.MatchEnded) {
	ws, _, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws", nil)
	if err != nil {
		log.Fatalf("[%s] 连接失败: %v", player, err)
	}
	defer ws.Close()
	mustSend(ws, &matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Join{Join: &matchpb.JoinRequest{Player: player}}})

	var last *matchpb.MatchEnded
	for last == nil {
		_, data, err := ws.ReadMessage()
		if err != nil {
			log.Fatalf("[%s] 读事件失败: %v", player, err)
		}
		var msg matchpb.ServerMessage
		if err := (protojson.UnmarshalOptions{}).Unmarshal(data, &msg); err != nil {
			log.Fatalf("[%s] 解析失败: %v", player, err)
		}
		if env := msg.GetEvent(); env != nil {
			if pr := env.GetPanelReady(); pr != nil {
				code, ok := solutions[templateIDOf(pr.GetProblem().GetId())]
				if !ok {
					log.Fatalf("未知题目 %s（smoke 的解表来自 templates.All，模板缺失？）", pr.GetProblem().GetId())
				}
				mustSend(ws, &matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Submit{Submit: &matchpb.SubmitRequest{Code: code}}})
			}
			if me := env.GetMatchEnded(); me != nil {
				last = me
			}
		}
	}
	done <- last
}

func mustSend(ws *websocket.Conn, msg *matchpb.ClientMessage) {
	b, err := protojson.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}
	if err := ws.WriteMessage(websocket.TextMessage, b); err != nil {
		log.Fatal(err)
	}
}
