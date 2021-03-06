package stream_chat // nolint: golint

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CreateChannel(t *testing.T) {
	c := initClient(t)

	t.Run("get existing channel", func(t *testing.T) {
		ch := initChannel(t, c)
		got, err := c.CreateChannel(ch.Type, ch.ID, serverUser.ID, nil)
		mustNoError(t, err, "create channel", ch)

		assert.Equal(t, c, got.client, "client link")
		assert.Equal(t, ch.Type, got.Type, "channel type")
		assert.Equal(t, ch.ID, got.ID, "channel id")
		assert.Equal(t, got.MemberCount, ch.MemberCount, "member count")
		assert.Len(t, got.Members, got.MemberCount, "members length")
	})

	tests := []struct {
		name    string
		_type   string
		id      string
		userID  string
		data    map[string]interface{}
		wantErr bool
	}{
		{"create channel with ID", "messaging", randomString(12), serverUser.ID, nil, false},
		{"create channel without ID and members", "messaging", "", serverUser.ID, nil, true},
		{"create channel without ID but with members", "messaging", "", serverUser.ID,
			map[string]interface{}{"members": []string{testUsers[0].ID, testUsers[1].ID}}, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.CreateChannel(tt._type, tt.id, tt.userID, tt.data)
			if tt.wantErr {
				mustError(t, err, "create channel", tt)
				return
			}

			mustNoError(t, err, "create channel", tt)

			assert.Equal(t, tt._type, got.Type, "channel type")
			assert.NotEmpty(t, got.ID)
			if tt.id != "" {
				assert.Equal(t, tt.id, got.ID, "channel id")
			}
			assert.Equal(t, tt.userID, got.CreatedBy.ID, "channel created by")
		})
	}
}

func TestChannel_AddMembers(t *testing.T) {
	c := initClient(t)

	chanID := randomString(12)

	ch, err := c.CreateChannel("messaging", chanID, serverUser.ID, nil)
	mustNoError(t, err, "create channel")
	defer func() {
		mustNoError(t, ch.Delete(), "delete channel")
	}()

	assert.Empty(t, ch.Members, "members are empty")

	user := randomUser()

	err = ch.AddMembers(
		[]string{user.ID},
		&Message{Text: "some members", User: &User{ID: user.ID}},
	)
	mustNoError(t, err, "add members")

	// refresh channel state
	mustNoError(t, ch.refresh(), "refresh channel")

	assert.Equal(t, user.ID, ch.Members[0].User.ID, "members contain user id")
}

func TestChannel_InviteMembers(t *testing.T) {
	c := initClient(t)

	chanID := randomString(12)

	ch, err := c.CreateChannel("messaging", chanID, serverUser.ID, nil)
	mustNoError(t, err, "create channel")
	defer func() {
		mustNoError(t, ch.Delete(), "delete channel")
	}()

	assert.Empty(t, ch.Members, "members are empty")

	user := randomUser()

	err = ch.InviteMembers(user.ID)
	mustNoError(t, err, "invite members")

	// refresh channel state
	mustNoError(t, ch.refresh(), "refresh channel")

	assert.Equal(t, user.ID, ch.Members[0].User.ID, "members contain user id")
	assert.Equal(t, true, ch.Members[0].Invited, "member is invited")
	assert.Nil(t, ch.Members[0].InviteAcceptedAt, "invite is not accepted")
	assert.Nil(t, ch.Members[0].InviteRejectedAt, "invite is not rejected")
}

func TestChannel_Moderation(t *testing.T) {
	c := initClient(t)

	// init random channel
	chanID := randomString(12)
	ch, err := c.CreateChannel("messaging", chanID, serverUser.ID, nil)
	mustNoError(t, err, "create channel")
	defer func() {
		mustNoError(t, ch.Delete(), "delete channel")
	}()

	assert.Empty(t, ch.Members, "members are empty")

	user := randomUser()

	err = ch.AddModeratorsWithMessage(
		[]string{user.ID},
		&Message{Text: "accepted", User: &User{ID: user.ID}},
	)

	mustNoError(t, err, "add moderators")

	// refresh channel state
	mustNoError(t, ch.refresh(), "refresh channel")

	assert.Equal(t, user.ID, ch.Members[0].User.ID, "user exists")
	assert.Equal(t, "moderator", ch.Members[0].Role, "user role is moderator")

	err = ch.DemoteModerators(user.ID)
	mustNoError(t, err, "demote moderators")

	// refresh channel state
	mustNoError(t, ch.refresh(), "refresh channel")

	assert.Equal(t, user.ID, ch.Members[0].User.ID, "user exists")
	assert.Equal(t, "member", ch.Members[0].Role, "user role is member")
}

func TestChannel_BanUser(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)
	defer func() {
		mustNoError(t, ch.Delete(), "delete channel")
	}()

	user := randomUser()

	err := ch.BanUser(user.ID, serverUser.ID, nil)
	mustNoError(t, err, "ban user")

	err = ch.BanUser(user.ID, serverUser.ID, map[string]interface{}{
		"timeout": 3600,
		"reason":  "offensive language is not allowed here",
	})
	mustNoError(t, err, "ban user")

	err = ch.UnBanUser(user.ID, nil)
	mustNoError(t, err, "unban user")
}

func TestChannel_Delete(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)

	mustNoError(t, ch.Delete(), "delete channel")
}

func TestChannel_GetReplies(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)
	defer func() {
		mustNoError(t, ch.Delete(), "delete channel")
	}()

	user := randomUser()

	msg := &Message{Text: "test message"}

	msg, err := ch.SendMessage(msg, user.ID)
	mustNoError(t, err, "send message")

	reply := &Message{Text: "test reply", ParentID: msg.ID, Type: MessageTypeReply}
	_, err = ch.SendMessage(reply, serverUser.ID)
	mustNoError(t, err, "send reply")

	replies, err := ch.GetReplies(msg.ID, nil)
	mustNoError(t, err, "get replies")
	assert.Len(t, replies, 1)
}

func TestChannel_MarkRead(t *testing.T) {

}

func TestChannel_RemoveMembers(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)
	defer func() {
		mustNoError(t, ch.Delete(), "delete channel")
	}()

	user := randomUser()
	err := ch.RemoveMembers(
		[]string{user.ID},
		&Message{Text: "some members", User: &User{ID: user.ID}},
	)

	mustNoError(t, err, "remove members")

	for _, member := range ch.Members {
		assert.NotEqual(t, member.User.ID, user.ID, "member is not present")
	}
}

func TestChannel_SendEvent(t *testing.T) {

}

func TestChannel_SendMessage(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)
	defer func() {
		mustNoError(t, ch.Delete(), "delete channel")
	}()

	user := randomUser()
	msg := &Message{
		Text: "test message",
		User: user,
	}

	msg, err := ch.SendMessage(msg, serverUser.ID)
	mustNoError(t, err, "send message")
	// check that message was updated
	assert.NotEmpty(t, msg.ID, "message has ID")
	assert.NotEmpty(t, msg.HTML, "message has HTML body")
}

func TestChannel_Truncate(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)
	defer func() {
		mustNoError(t, ch.Delete(), "delete channel")
	}()

	user := randomUser()
	msg := &Message{
		Text: "test message",
		User: user,
	}
	msg, err := ch.SendMessage(msg, serverUser.ID)
	mustNoError(t, err, "send message")

	// refresh channel state
	mustNoError(t, ch.refresh(), "refresh channel")

	assert.Equal(t, ch.Messages[0].ID, msg.ID, "message exists")

	err = ch.Truncate()
	mustNoError(t, err, "truncate channel")

	// refresh channel state
	mustNoError(t, ch.refresh(), "refresh channel")

	assert.Empty(t, ch.Messages, "message not exists")
}

func TestChannel_Update(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)

	err := ch.Update(map[string]interface{}{"color": "blue"},
		&Message{Text: "color is blue", User: &User{ID: testUsers[0].ID}})
	require.NoError(t, err)
}

func TestChannel_AddModerators(t *testing.T) {

}

func TestChannel_DemoteModerators(t *testing.T) {

}

func TestChannel_UnBanUser(t *testing.T) {

}

func TestChannel_SendFile(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)

	var url string

	t.Run("Send file", func(t *testing.T) {
		file, err := os.Open(path.Join("testdata", "helloworld.txt"))
		if err != nil {
			t.Fatal(err)
		}

		url, err = ch.SendFile(SendFileRequest{
			Reader:   file,
			FileName: "HelloWorld.txt",
			User:     randomUser(),
		})
		if err != nil {
			t.Fatalf("send file failed: %s", err)
		}
		if url == "" {
			t.Fatal("upload file returned empty url")
		}
	})

	t.Run("Delete file", func(t *testing.T) {
		err := ch.DeleteFile(url)
		if err != nil {
			t.Fatalf("delete file failed: %s", err.Error())
		}
	})
}

func TestChannel_SendImage(t *testing.T) {
	c := initClient(t)
	ch := initChannel(t, c)

	var url string

	t.Run("Send image", func(t *testing.T) {
		file, err := os.Open(path.Join("testdata", "helloworld.jpg"))
		if err != nil {
			t.Fatal(err)
		}

		url, err = ch.SendImage(SendFileRequest{
			Reader:      file,
			FileName:    "HelloWorld.jpg",
			User:        randomUser(),
			ContentType: "image/jpeg",
		})

		if err != nil {
			t.Fatalf("Send image failed: %s", err.Error())
		}

		if url == "" {
			t.Fatal("upload image returned empty url")
		}
	})

	t.Run("Delete image", func(t *testing.T) {
		err := ch.DeleteImage(url)
		if err != nil {
			t.Fatalf("delete image failed: %s", err.Error())
		}
	})
}

func TestChannel_AcceptInvite(t *testing.T) {
	c := initClient(t)

	_, err := c.UpdateUsers(testUsers...)
	mustNoError(t, err, "update users")

	members := make([]string, 0, len(testUsers))
	for i := range testUsers {
		members = append(members, testUsers[i].ID)
	}

	ch, err := c.CreateChannel("team", randomString(12), serverUser.ID, map[string]interface{}{
		"members": members,
		"invites": []string{members[0]},
	})

	require.NoError(t, err, "create channel")
	err = ch.AcceptInvite(members[0], &Message{Text: "accepted", User: &User{ID: members[0]}})
	require.NoError(t, err, "accept invite")
}

func TestChannel_RejectInvite(t *testing.T) {
	c := initClient(t)

	_, err := c.UpdateUsers(testUsers...)
	mustNoError(t, err, "update users")

	members := make([]string, 0, len(testUsers))
	for i := range testUsers {
		members = append(members, testUsers[i].ID)
	}

	ch, err := c.CreateChannel("team", randomString(12), serverUser.ID, map[string]interface{}{
		"members": members,
		"invites": []string{members[0]},
	})

	require.NoError(t, err, "create channel")
	err = ch.RejectInvite(members[0], &Message{Text: "rejected", User: &User{ID: members[0]}})
	require.NoError(t, err, "reject invite")
}
