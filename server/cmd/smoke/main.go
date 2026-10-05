// smoke：headless 端到端冒烟客户端——两个 WS 客户端接入同一场本地对局并打满。
// 用法：先起 judge 与 codewar 服务，再运行
//
//	go run ./cmd/smoke
//
// 03 号票链路（裁决 9）：双方各解一道不同行动题 → 按序结算——
//   - 奇数回合：P1 解移动题拉开距离（(3,2)↔(3,0)），P2 解攻击题 → 攻击结算时
//     P1 已不在相邻 4 格 → 闪避（无伤害）
//   - 偶数回合：P1 解移动题回到 (3,2)（重新进入相邻格），P2 解攻击题 → 命中 22
//
// P2 于第 2/4/6/8/10 回合各命中一次（5×22=110 ≥ 100）→ 第 10 回合 KO，
// P2 获胜。全链路验证：三槽面板、私有题判题、逐动作即时结算、位置命中判定。
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
// 同一模板的参考解可解任意实例）。
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

// plan：P1 的移动脚本——奇数回合拉到 (3,0)，偶数回合回 (3,2)。
// P2 始终解攻击题。回合号从 1 起。
func moveTarget(round int) *matchpb.Position {
	if round%2 == 1 {
		return &matchpb.Position{Row: 3, Col: 0}
	}
	return &matchpb.Position{Row: 3, Col: 2}
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
	if a.Reason != "ko" || a.Winner != "p2" {
		log.Fatalf("预期 P2 于回合上限内 KO（P1 移动/攻击链），实际 %q/%q", a.Winner, a.Reason)
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

	round := 0
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
			if rs := env.GetRoundStarted(); rs != nil {
				round = int(rs.GetRound())
			}
			if pr := env.GetPanelReady(); pr != nil {
				var action, problemID string
				var moveTo *matchpb.Position
				if player == "p1" {
					// P1 全程解移动题（奇偶回合在 (3,0)/(3,2) 间往返）
					action = "move"
					problemID = pr.GetP1().GetMove().GetId()
					moveTo = moveTarget(round)
				} else {
					// P2 全程解攻击题
					action = "attack"
					problemID = pr.GetAttack().GetId()
				}
				code, ok := solutions[templateIDOf(problemID)]
				if !ok {
					log.Fatalf("未知题目 %s（smoke 的解表来自 templates.All，模板缺失？）", problemID)
				}
				mustSend(ws, &matchpb.ClientMessage{Payload: &matchpb.ClientMessage_Submit{Submit: &matchpb.SubmitRequest{
					Code: code, Action: action, MoveTo: moveTo,
				}}})
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
