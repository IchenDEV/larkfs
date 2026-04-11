package vfs_test

import (
	"context"
	"strings"
	"testing"

	"github.com/IchenDEV/larkfs/pkg/vfs"
	"github.com/IchenDEV/larkfs/tests/testutil"
)

func TestVFSDomainContentBlackbox(t *testing.T) {
	runner := &testutil.Runner{RunFn: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "wiki spaces list"):
			return []byte(`{"data":{"items":[{"space_id":"sp1","name":"Space"}]}}`), nil
		case strings.HasPrefix(joined, "wiki nodes list"):
			return []byte(`{"data":{"items":[{"node_token":"node1","title":"Doc","obj_type":"docx"}]}}`), nil
		case strings.HasPrefix(joined, "wiki spaces get_node"):
			return []byte(`{"data":{"node":{"obj_type":"docx","obj_token":"doc1"}}}`), nil
		case strings.HasPrefix(joined, "docs +fetch"):
			return []byte(`{"data":{"markdown":"# Wiki\n\n- [ ] follow up"}}`), nil
		case strings.HasPrefix(joined, "docs +update"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "im chats list"):
			return []byte(`{"data":{"items":[{"chat_id":"chat1","name":"Chat"}]}}`), nil
		case strings.HasPrefix(joined, "im +chat-messages-list"):
			return []byte(`{"data":{"messages":[{"msg_type":"text","message_id":"msg1","content":"Hi","sender":{"name":"Alice"}}]}}`), nil
		case strings.HasPrefix(joined, "im +messages-send"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "mail +triage"):
			return []byte(`{"data":[{"message_id":"m1","from":"alice","subject":"Hello","date":"2026-04-11"}]}`), nil
		case strings.HasPrefix(joined, "mail +message"):
			return []byte(`{"data":{"message_id":"m1","thread_id":"t1","from":"alice","to":["bob"],"cc":[],"date":"2026-04-11","subject":"Hello","body":"Body","labels":["INBOX"]}}`), nil
		case strings.HasPrefix(joined, "mail user_mailbox.messages trash"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "calendar +agenda"):
			return []byte(`{"data":[{"event_id":"evt1","summary":"Sync","start_time":"start","end_time":"end"}]}`), nil
		case strings.HasPrefix(joined, "calendar +create"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "api GET /open-apis/task/v2/tasks/task1"):
			return []byte(`{"data":{"task":{"task_id":"task1","summary":"Task","status":"todo"}}}`), nil
		case strings.HasPrefix(joined, "api GET /open-apis/task/v2/tasks"):
			return []byte(`{"data":{"items":[{"task_id":"task1","summary":"Task","status":"todo"}]}}`), nil
		case strings.HasPrefix(joined, "api POST /open-apis/task/v2/tasks"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "vc +notes"):
			return []byte(`{"data":{"items":[{"note_doc_token":"note","verbatim_doc_token":"verb"}]}}`), nil
		case strings.HasPrefix(joined, "vc +search"):
			return []byte(`{"data":{"items":[{"meeting_id":"meet1","topic":"Review","start_time":"2026-04-11"}]}}`), nil
		case strings.HasPrefix(joined, "vc meeting get"):
			return []byte(`{"data":{"meeting_id":"meet1"}}`), nil
		case strings.HasPrefix(joined, "vc meeting.recording get"):
			return []byte(`{"data":{"recording":{"url":"https://example.test/r.mp4"}}}`), nil
		case strings.HasPrefix(joined, "drive +download"):
			return []byte("recording"), nil
		default:
			t.Fatalf("unexpected args: %v", args)
			return nil, nil
		}
	}}
	ops := newFullTestOps(t, runner)

	if _, err := ops.ReadDir(context.Background(), "/wiki"); err != nil {
		t.Fatalf("ReadDir(wiki) error: %v", err)
	}
	if _, err := ops.ReadDir(context.Background(), "/wiki/Space"); err != nil {
		t.Fatalf("ReadDir(wiki space) error: %v", err)
	}
	if data, err := ops.Read(context.Background(), "/wiki/Space/Doc.md"); err != nil || !strings.Contains(string(data), "Wiki") {
		t.Fatalf("Read(wiki) = %s, %v", data, err)
	}
	if err := ops.Write(context.Background(), "/wiki/Space/Doc.md", []byte("updated")); err != nil {
		t.Fatalf("Write(wiki) error: %v", err)
	}

	if _, err := ops.ReadDir(context.Background(), "/im"); err != nil {
		t.Fatalf("ReadDir(im) error: %v", err)
	}
	if _, err := ops.ReadDir(context.Background(), "/im/Chat"); err != nil {
		t.Fatalf("ReadDir(im chat) error: %v", err)
	}
	if data, err := ops.Read(context.Background(), "/im/Chat/latest.md"); err != nil || !strings.Contains(string(data), "Alice") {
		t.Fatalf("Read(im latest) = %s, %v", data, err)
	}
	if err := ops.Write(context.Background(), "/im/Chat/_send.md", []byte("hello")); err != nil {
		t.Fatalf("Write(im send) error: %v", err)
	}
	if err := ops.Write(context.Background(), "/im/Chat/latest.md", []byte("nope")); err == nil {
		t.Fatal("Write(im latest) expected read-only error")
	}

	inbox := &vfs.VNode{Name: "INBOX", Token: "INBOX", NodeType: vfs.NodeDir, Kind: vfs.NodeKindResource, Domain: "mail"}
	ops.Tree().DomainNode("mail").AddChild(inbox)
	children, err := ops.ReadDir(context.Background(), "/mail/INBOX")
	if err != nil {
		t.Fatalf("ReadDir(mail) = %+v, %v", children, err)
	}
	var mailName string
	for _, child := range children {
		if child.Kind == vfs.NodeKindResource {
			mailName = child.Name
			break
		}
	}
	if mailName == "" {
		t.Fatalf("expected mail resource child, got %+v", children)
	}
	if data, err := ops.Read(context.Background(), "/mail/INBOX/"+mailName); err != nil || !strings.Contains(string(data), "Body") {
		t.Fatalf("Read(mail) = %s, %v", data, err)
	}
	if err := ops.Remove(context.Background(), "/mail/INBOX/"+mailName); err != nil {
		t.Fatalf("Remove(mail) error: %v", err)
	}
	if inbox.GetChild(mailName) != nil {
		t.Fatal("mail child should be removed from VFS cache")
	}

	if data, err := ops.Read(context.Background(), "/calendar/_create.md"); err != nil || !strings.Contains(string(data), "New Event") {
		t.Fatalf("Read(calendar create) = %q, %v", data, err)
	}
	if err := ops.Write(context.Background(), "/calendar/_create.md", []byte("event")); err != nil {
		t.Fatalf("Write(calendar create) error: %v", err)
	}
	if _, err := ops.ReadDir(context.Background(), "/calendar"); err != nil {
		t.Fatalf("ReadDir(calendar) error: %v", err)
	}
	if data, err := ops.Read(context.Background(), "/calendar/Sync.md"); err != nil || !strings.Contains(string(data), "Sync") {
		t.Fatalf("Read(calendar event) = %s, %v", data, err)
	}
	if err := ops.Write(context.Background(), "/tasks/_create.md", []byte(`{"summary":"Task"}`)); err != nil {
		t.Fatalf("Write(task create) error: %v", err)
	}
	if _, err := ops.ReadDir(context.Background(), "/tasks"); err != nil {
		t.Fatalf("ReadDir(tasks) error: %v", err)
	}
	if data, err := ops.Read(context.Background(), "/tasks/Task.md"); err != nil || !strings.Contains(string(data), "Task") {
		t.Fatalf("Read(task) = %q, %v", data, err)
	}

	for _, tc := range []struct {
		name string
		part string
		want string
	}{
		{"_meta.json", "meta", "meet1"},
		{"summary.md", "summary", "Wiki"},
		{"transcript.md", "transcript", "Wiki"},
		{"todos.md", "todos", "follow up"},
		{"recording.mp4", "recording", "recording"},
	} {
		node := &vfs.VNode{Name: tc.name, Token: "meet1|" + tc.part, NodeType: vfs.NodeFile, Kind: vfs.NodeKindResource, Domain: "meetings"}
		ops.Tree().DomainNode("meetings").AddChild(node)
		data, err := ops.Read(context.Background(), "/meetings/"+tc.name)
		if err != nil || !strings.Contains(string(data), tc.want) {
			t.Fatalf("Read(%s) = %q, %v", tc.name, data, err)
		}
	}
	dateDirs, err := ops.ReadDir(context.Background(), "/meetings")
	if err != nil || len(dateDirs) == 0 {
		t.Fatalf("ReadDir(meetings) = %+v, %v", dateDirs, err)
	}
	if meetings, err := ops.ReadDir(context.Background(), "/meetings/"+dateDirs[0].Name); err != nil || len(meetings) == 0 {
		t.Fatalf("ReadDir(meetings date) = %+v, %v", meetings, err)
	}
}
