package adapter_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IchenDEV/larkfs/pkg/adapter"
	"github.com/IchenDEV/larkfs/pkg/cache"
	"github.com/IchenDEV/larkfs/pkg/doctype"
	"github.com/IchenDEV/larkfs/tests/testutil"
)

func TestDriveAdapterRoutesAndCaches(t *testing.T) {
	runner := &testutil.Runner{RunFn: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "drive files list"):
			return []byte(`{"data":{"files":[{"token":"doc_1","name":"Doc","type":"docx"},{"token":"fld_1","name":"Folder","type":"folder"}],"has_more":true,"next_page_token":"n"}}`), nil
		case strings.HasPrefix(joined, "docs +fetch"):
			return []byte(`{"data":{"markdown":"# Doc"}}`), nil
		case strings.HasPrefix(joined, "sheets +read"):
			return []byte(`{"data":{"valueRange":{"values":[["A"]]}}}`), nil
		case strings.HasPrefix(joined, "sheets +info"):
			return []byte(`{"data":{"sheets":{"sheets":[{"sheet_id":"s1","title":"Sheet One"}]}}}`), nil
		case strings.HasPrefix(joined, "base +record-upsert"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "docs +create"):
			return []byte(`{"data":{"doc_id":"doc_new"}}`), nil
		case strings.HasPrefix(joined, "drive files delete"):
			return []byte(`{"code":0}`), nil
		default:
			t.Fatalf("unexpected args: %v", args)
			return nil, nil
		}
	}}
	meta, namer, registry := testutil.NewDeps(t, runner)
	a := adapter.NewDriveAdapter(runner, registry, meta, namer)

	list, err := a.ListRoot(context.Background())
	if err != nil || len(list.Entries) != 2 || list.Entries[0].Name != "Doc.md" || !list.Page.HasMore {
		t.Fatalf("ListRoot() = %+v, %v", list, err)
	}
	before := len(runner.Calls)
	if _, err := a.ListFolder(context.Background(), ""); err != nil {
		t.Fatalf("cached ListFolder() error: %v", err)
	}
	if len(runner.Calls) != before {
		t.Fatal("expected ListFolder cache hit not to call runner")
	}
	typed, err := a.ListByType(context.Background(), "shtcn1", doctype.TypeSheet)
	if err != nil || len(typed.Entries) != 2 || typed.Entries[1].Token != "shtcn1|s1" {
		t.Fatalf("ListByType(sheet) = %+v, %v", typed, err)
	}
	before = len(runner.Calls)
	if _, err := a.ListByType(context.Background(), "shtcn1", doctype.TypeSheet); err != nil {
		t.Fatalf("cached ListByType() error: %v", err)
	}
	if len(runner.Calls) != before {
		t.Fatal("expected ListByType cache hit not to call runner")
	}
	data, err := a.Read(context.Background(), "doc_1", doctype.TypeDocx)
	if err != nil || string(data) != "# Doc" {
		t.Fatalf("Read(docx) = %q, %v", data, err)
	}
	csvData, err := a.Read(context.Background(), "shtcn1|s1", doctype.TypeFile)
	if err != nil || string(csvData) != "A\n" {
		t.Fatalf("Read(composite sheet) = %q, %v", csvData, err)
	}
	if err := a.Write(context.Background(), "bascn1|tbl1", doctype.TypeFile, []byte(`{"id":"1"}`+"\n")); err != nil {
		t.Fatalf("Write(composite base) error: %v", err)
	}
	token, err := a.Create(context.Background(), "folder", "Doc", doctype.TypeDocx, nil)
	if err != nil || token != "doc_new" {
		t.Fatalf("Create() = %q, %v", token, err)
	}
	if err := a.Delete(context.Background(), "doc_1", doctype.TypeDocx); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestCalendarTaskMailWikiIMAndMeetingAdapters(t *testing.T) {
	runner := &testutil.Runner{RunFn: func(_ context.Context, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "calendar +agenda"):
			return []byte(`{"data":[{"event_id":"evt_1","summary":"Team Sync","start_time":"2026-04-11T10:00:00","end_time":"2026-04-11T11:00:00","location":"Room"}]}`), nil
		case strings.HasPrefix(joined, "calendar +create"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "api GET /open-apis/task/v2/tasks/task_1"):
			return []byte(`{"data":{"task":{"task_id":"task_1","summary":"Ship","due":"today","status":"todo"}}}`), nil
		case strings.HasPrefix(joined, "api GET /open-apis/task/v2/tasks"):
			return []byte(`{"data":{"items":[{"task_id":"task_1","summary":"Ship","due":"today","status":"todo"}]}}`), nil
		case strings.HasPrefix(joined, "api POST /open-apis/task/v2/tasks"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "mail user_mailbox.folders list"):
			return []byte(`{"data":{"items":[{"folder_id":"inbox","name":"INBOX"}]}}`), nil
		case strings.HasPrefix(joined, "mail +triage"):
			return []byte(`{"data":[{"message_id":"m1","from":"alice","subject":"Hello","date":"2026-04-11T00:00:00Z"}]}`), nil
		case strings.HasPrefix(joined, "mail +message"):
			return []byte(`{"data":{"message_id":"m1","thread_id":"t1","from":"alice","to":["bob"],"cc":[],"date":"2026-04-11","subject":"Hello","body":"Body","labels":["INBOX"]}}`), nil
		case strings.HasPrefix(joined, "mail +send"), strings.HasPrefix(joined, "mail +reply"), strings.HasPrefix(joined, "mail user_mailbox.messages trash"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "wiki spaces list"):
			return []byte(`{"data":{"items":[{"space_id":"sp1","name":"Product"}],"has_more":true,"page_token":"next"}}`), nil
		case strings.HasPrefix(joined, "wiki nodes list"):
			return []byte(`{"data":{"items":[{"node_token":"node1","title":"Spec","obj_type":"docx","has_child":false}],"has_more":false}}`), nil
		case strings.HasPrefix(joined, "wiki spaces get_node"):
			return []byte(`{"data":{"node":{"obj_type":"docx","obj_token":"doc_1"}}}`), nil
		case strings.HasPrefix(joined, "docs +fetch"):
			return []byte(`{"data":{"markdown":"# Spec"}}`), nil
		case strings.HasPrefix(joined, "docs +update"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "im chats list"):
			return []byte(`{"data":{"items":[{"chat_id":"chat1","name":"Team"}],"has_more":true,"page_token":"next"}}`), nil
		case strings.HasPrefix(joined, "im +chat-messages-list"):
			return []byte(`{"data":{"messages":[{"msg_type":"text","message_id":"msg1","content":"Hi","create_time":"now","sender":{"name":"Alice"}},{"msg_type":"image","message_id":"img1","content":"{}"}]}}`), nil
		case strings.HasPrefix(joined, "im +messages-send"):
			return []byte(`{"ok":true}`), nil
		case strings.HasPrefix(joined, "vc +search"):
			return []byte(`{"data":{"items":[{"meeting_id":"meet1","topic":"Review","start_time":"2026-04-11"}]}}`), nil
		case strings.HasPrefix(joined, "vc meeting get"):
			return []byte(`{"data":{"meeting_id":"meet1"}}`), nil
		case strings.HasPrefix(joined, "vc +notes"):
			return []byte(`{"data":{"items":[{"note_doc_token":"note_doc","verbatim_doc_token":"transcript_doc"}]}}`), nil
		case strings.HasPrefix(joined, "vc meeting.recording get"):
			return []byte(`{"data":{"recording":{"url":"https://example.com/r.mp4"}}}`), nil
		case strings.HasPrefix(joined, "drive +download"):
			return []byte("video"), nil
		default:
			t.Fatalf("unexpected args: %v", args)
			return nil, nil
		}
	}}
	meta, namer, registry := testutil.NewDeps(t, runner)

	cal := adapter.NewCalendarAdapter(runner, meta, namer)
	events, err := cal.ListEvents(context.Background())
	if err != nil || len(events.Entries) != 2 || events.Entries[0].Name != "Team Sync.md" {
		t.Fatalf("ListEvents() = %+v, %v", events, err)
	}
	event, err := cal.ReadEvent(context.Background(), "evt_1")
	if err != nil || !strings.Contains(string(event), "Team Sync") {
		t.Fatalf("ReadEvent() = %s, %v", event, err)
	}
	if err := cal.CreateEvent(context.Background(), []byte("new event")); err != nil {
		t.Fatalf("CreateEvent() error: %v", err)
	}

	task := adapter.NewTaskAdapter(runner, meta, namer)
	tasks, err := task.ListTasks(context.Background())
	if err != nil || len(tasks.Entries) != 2 || tasks.Entries[0].Name != "Ship.md" {
		t.Fatalf("ListTasks() = %+v, %v", tasks, err)
	}
	taskBody, err := task.ReadTask(context.Background(), "task_1")
	if err != nil || !strings.Contains(string(taskBody), "Ship") {
		t.Fatalf("ReadTask() = %s, %v", taskBody, err)
	}
	if err := task.CreateTask(context.Background(), []byte(`{"summary":"Ship"}`)); err != nil {
		t.Fatalf("CreateTask() error: %v", err)
	}

	mail := adapter.NewMailAdapter(runner, meta, namer)
	folders, err := mail.ListFolders(context.Background())
	if err != nil || len(folders.Entries) != 3 {
		t.Fatalf("ListFolders() = %+v, %v", folders, err)
	}
	msgs, err := mail.ListMessages(context.Background(), "INBOX")
	if err != nil || len(msgs.Entries) != 1 || !strings.HasSuffix(msgs.Entries[0].Name, ".md") {
		t.Fatalf("ListMessages() = %+v, %v", msgs, err)
	}
	msg, err := mail.ReadMessage(context.Background(), "m1")
	if err != nil || !strings.Contains(string(msg), "Body") {
		t.Fatalf("ReadMessage() = %s, %v", msg, err)
	}
	if err := mail.Send(context.Background(), "bob", "Hi", "Body"); err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	if err := mail.Reply(context.Background(), "m1", "Reply"); err != nil {
		t.Fatalf("Reply() error: %v", err)
	}
	if err := mail.Trash(context.Background(), "m1"); err != nil {
		t.Fatalf("Trash() error: %v", err)
	}

	wiki := adapter.NewWikiAdapter(runner, registry, meta, namer)
	spaces, err := wiki.ListSpaces(context.Background())
	if err != nil || len(spaces.Entries) != 1 || spaces.Page.NextCursor != "next" {
		t.Fatalf("ListSpaces() = %+v, %v", spaces, err)
	}
	nodes, err := wiki.ListNodes(context.Background(), "sp1")
	if err != nil || len(nodes.Entries) != 1 || nodes.Entries[0].Name != "Spec.md" {
		t.Fatalf("ListNodes() = %+v, %v", nodes, err)
	}
	read, err := wiki.Read(context.Background(), "node1")
	if err != nil || string(read) != "# Spec" {
		t.Fatalf("Read() = %q, %v", read, err)
	}
	if err := wiki.Write(context.Background(), "node1", []byte("updated")); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	im := adapter.NewIMAdapter(runner, meta, namer)
	chats, err := im.ListChats(context.Background())
	if err != nil || len(chats.Entries) != 1 || chats.Entries[0].Name != "Team" {
		t.Fatalf("ListChats() = %+v, %v", chats, err)
	}
	contents, err := im.ListChatContents(context.Background(), "chat1")
	if err != nil || len(contents.Entries) != 3 {
		t.Fatalf("ListChatContents() = %+v, %v", contents, err)
	}
	files, err := im.ListChatFiles(context.Background(), "chat1")
	if err != nil || len(files.Entries) != 1 || files.Entries[0].Name != "img1.png" {
		t.Fatalf("ListChatFiles() = %+v, %v", files, err)
	}
	messages, err := im.ReadMessages(context.Background(), "chat1")
	if err != nil || !strings.Contains(string(messages), "Alice") {
		t.Fatalf("ReadMessages() = %s, %v", messages, err)
	}
	if err := im.SendMessage(context.Background(), "chat1", []byte("hello")); err != nil {
		t.Fatalf("SendMessage() error: %v", err)
	}

	meeting := adapter.NewMeetingAdapter(runner, meta, namer, t.TempDir())
	if dates := meeting.ListDateDirs(); len(dates.Entries) != 30 {
		t.Fatalf("ListDateDirs() entries = %d", len(dates.Entries))
	}
	meetings, err := meeting.ListMeetings(context.Background(), "2026-04-11")
	if err != nil || len(meetings.Entries) != 1 || meetings.Entries[0].Name != "Review" {
		t.Fatalf("ListMeetings() = %+v, %v", meetings, err)
	}
	if contents := meeting.ListMeetingContents("meet1"); len(contents.Entries) != 5 {
		t.Fatalf("ListMeetingContents() entries = %d", len(contents.Entries))
	}
	if metaJSON, err := meeting.ReadMeta(context.Background(), "meet1"); err != nil || !strings.Contains(string(metaJSON), "meet1") {
		t.Fatalf("ReadMeta() = %s, %v", metaJSON, err)
	}
	if summary, err := meeting.ReadSummary(context.Background(), "meet1"); err != nil || string(summary) != "# Spec" {
		t.Fatalf("ReadSummary() = %q, %v", summary, err)
	}
	if transcript, err := meeting.ReadTranscript(context.Background(), "meet1"); err != nil || string(transcript) != "# Spec" {
		t.Fatalf("ReadTranscript() = %q, %v", transcript, err)
	}
	if recording, err := meeting.ReadRecording(context.Background(), "meet1"); err != nil || string(recording) != "video" {
		t.Fatalf("ReadRecording() = %q, %v", recording, err)
	}
}

func TestIMListChatFilesReturnsRunnerErrorFromBlackbox(t *testing.T) {
	want := errors.New("runner failed")
	im := adapter.NewIMAdapter(&testutil.Runner{
		RunFn: func(_ context.Context, args ...string) ([]byte, error) {
			return nil, want
		},
	}, cache.NewMetadataCache(time.Minute), nil)

	_, err := im.ListChatFiles(context.Background(), "chat-1")
	if !errors.Is(err, want) {
		t.Fatalf("ListChatFiles() error = %v, want %v", err, want)
	}
}
