package stream_chat

import (
	"errors"
	"net/url"
	"path"
	"time"
)

type ChannelMember struct {
	UserID      string `json:"user_id,omitempty"`
	User        *User  `json:"user,omitempty"`
	IsModerator bool   `json:"is_moderator,omitempty"`

	Invited          bool       `json:"invited,omitempty"`
	InviteAcceptedAt *time.Time `json:"invite_accepted_at,omitempty"`
	InviteRejectedAt *time.Time `json:"invite_rejected_at,omitempty"`
	Role             string     `json:"role,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Channel struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	CID  string `json:"cid"` // full id in format channel_type:channel_ID

	Config ChannelConfig `json:"config"`

	CreatedBy *User `json:"created_by"`
	Frozen    bool  `json:"frozen"`

	MemberCount int              `json:"member_count"`
	Members     []*ChannelMember `json:"members"`

	Messages []*Message `json:"messages"`
	Read     []*User    `json:"read"`

	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastMessageAt time.Time `json:"last_message_at"`

	client RestClient
}

type ChannelOptions struct {
	ID   string // channel ID, required
	Type string // channel type, for example "messaging"; required

	Data map[string]interface{} // extra channel data
}

// CreateChannel creates new channel of given type and id or returns already created one
func CreateChannel(client RestClient, options ChannelOptions, userID string) (*Channel, error) {
	switch {
	case options.Type == "":
		return nil, errors.New("channel type is empty")
	case options.ID == "":
		return nil, errors.New("channel ID is empty")
	case userID == "":
		return nil, errors.New("user ID is empty")
	}

	ch := &Channel{
		Type:      options.Type,
		ID:        options.ID,
		CreatedBy: &User{ID: userID},

		client: client,
	}

	args := map[string]interface{}{
		"watch":    false,
		"state":    true,
		"presence": false,
	}

	err := ch.query(args, options.Data)

	return ch, err
}

type queryResponse struct {
	Channel  *Channel         `json:"channel,omitempty"`
	Messages []*Message       `json:"messages,omitempty"`
	Members  []*ChannelMember `json:"members,omitempty"`
	Read     []*User          `json:"read,omitempty"`
}

func (q queryResponse) updateChannel(ch *Channel) {
	if q.Channel != nil {
		// save client pointer but update channel information
		client := ch.client
		*ch = *q.Channel
		ch.client = client
	}

	if q.Members != nil {
		ch.Members = q.Members
	}
	if q.Messages != nil {
		ch.Messages = q.Messages
	}
	if q.Read != nil {
		ch.Read = q.Read
	}
}

// query makes request to channel api and updates channel internal state
func (ch *Channel) query(options map[string]interface{}, data map[string]interface{}) (err error) {
	payload := map[string]interface{}{
		"state": true,
	}

	for k, v := range options {
		payload[k] = v
	}

	if data == nil {
		data = map[string]interface{}{}
	}

	data["created_by"] = map[string]interface{}{"id": ch.CreatedBy.ID}

	payload["data"] = data

	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID), "query")

	var resp queryResponse

	err = ch.client.Post(p, nil, payload, &resp)
	if err != nil {
		return err
	}

	resp.updateChannel(ch)

	return nil
}

// Update edits the channel's custom properties
//
// options: the object to update the custom properties of this channel with
// message: optional update message
func (ch *Channel) Update(options map[string]interface{}, message string) error {
	payload := map[string]interface{}{
		"data":    options,
		"message": message,
	}

	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID))

	return ch.client.Post(p, nil, payload, nil)
}

// Delete removes the channel. Messages are permanently removed.
func (ch *Channel) Delete() error {
	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID))

	return ch.client.Delete(p, nil, nil)
}

// Truncate removes all messages from the channel
func (ch *Channel) Truncate() error {
	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID), "truncate")

	return ch.client.Post(p, nil, nil, nil)
}

// AddMembers adds members with given user IDs to the channel
func (ch *Channel) AddMembers(userIDs ...string) error {
	if len(userIDs) == 0 {
		return errors.New("user IDs are empty")
	}

	data := map[string]interface{}{
		"add_members": userIDs,
	}

	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID))

	return ch.client.Post(p, nil, data, nil)
}

//  RemoveMembers deletes members with given IDs from the channel
func (ch *Channel) RemoveMembers(userIDs ...string) error {
	if len(userIDs) == 0 {
		return errors.New("user IDs are empty")
	}

	data := map[string]interface{}{
		"remove_members": userIDs,
	}

	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID))

	var resp queryResponse

	err := ch.client.Post(p, nil, data, &resp)
	if err != nil {
		return err
	}

	resp.updateChannel(ch)

	return nil
}

// AddModerators adds moderators with given IDs to the channel
func (ch *Channel) AddModerators(userIDs ...string) error {
	if len(userIDs) == 0 {
		return errors.New("user IDs are empty")
	}

	data := map[string]interface{}{
		"add_moderators": userIDs,
	}

	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID))

	return ch.client.Post(p, nil, data, nil)
}

// DemoteModerators moderators with given IDs from the channel
func (ch *Channel) DemoteModerators(userIDs ...string) error {
	if len(userIDs) == 0 {
		return errors.New("user IDs are empty")
	}

	data := map[string]interface{}{
		"demote_moderators": userIDs,
	}

	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID))

	return ch.client.Post(p, nil, data, nil)
}

//  MarkRead send the mark read event for user with given ID, only works if the `read_events` setting is enabled
//  options: additional data, ie {"messageID": last_messageID}
func (ch *Channel) MarkRead(userID string, options map[string]interface{}) error {
	switch {
	case userID == "":
		return errors.New("user ID must be not empty")
	case options == nil:
		options = map[string]interface{}{}
	}

	p := path.Join("channels", url.PathEscape(ch.Type), url.PathEscape(ch.ID), "read")

	options["user"] = map[string]interface{}{"id": userID}

	return ch.client.Post(p, nil, options, nil)
}

// BanUser bans target user ID from this channel
// userID: user who bans target
// options: additional ban options, ie {"timeout": 3600, "reason": "offensive language is not allowed here"}
func (ch *Channel) BanUser(targetID string, userID string, options map[string]interface{}) error {
	switch {
	case targetID == "":
		return errors.New("target ID is empty")
	case userID == "":
		return errors.New("user ID is empty")
	case options == nil:
		options = map[string]interface{}{}
	}

	options["type"] = ch.Type
	options["id"] = ch.ID
	options["target_user_id"] = targetID
	options["user_id"] = userID

	return ch.client.Post("moderation/ban", nil, options, nil)
}

// UnBanUser removes the ban for target user ID on this channel
func (ch *Channel) UnBanUser(targetID string, options map[string]string) error {
	switch {
	case targetID == "":
		return errors.New("target ID must be not empty")
	case options == nil:
		options = make(map[string]string, 3)
	}

	options["type"] = ch.Type
	options["id"] = ch.ID
	options["target_user_id"] = targetID

	params := make(map[string][]string, 3)
	for k, v := range options {
		params[k] = []string{v}
	}

	return ch.client.Delete("moderation/ban", params, nil)
}

// todo: cleanup this
func (ch *Channel) refresh() error {
	options := map[string]interface{}{
		"watch":    false,
		"state":    true,
		"presence": false,
	}

	err := ch.query(options, nil)

	return err
}
