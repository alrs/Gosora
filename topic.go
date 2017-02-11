package main
//import "fmt"
import "sync"
import "html/template"
import "database/sql"

type Topic struct
{
	ID int
	Title string
	Content string
	CreatedBy int
	Is_Closed bool
	Sticky bool
	CreatedAt string
	LastReplyAt string
	ParentID int
	Status string // Deprecated. Marked for removal.
	IpAddress string
	PostCount int
	LikeCount int
}

type TopicUser struct
{
	ID int
	Title string
	Content string
	CreatedBy int
	Is_Closed bool
	Sticky bool
	CreatedAt string
	LastReplyAt string
	ParentID int
	Status string // Deprecated. Marked for removal.
	IpAddress string
	PostCount int
	LikeCount int
	
	CreatedByName string
	Group int
	Avatar string
	Css template.CSS
	ContentLines int
	Tag string
	URL string
	URLPrefix string
	URLName string
	Level int
	Liked bool
}

type TopicsRow struct
{
	ID int
	Title string
	Content string
	CreatedBy int
	Is_Closed bool
	Sticky bool
	CreatedAt string
	LastReplyAt string
	ParentID int
	Status string // Deprecated. Marked for removal.
	IpAddress string
	PostCount int
	LikeCount int
	
	CreatedByName string
	Avatar string
	Css template.CSS
	ContentLines int
	Tag string
	URL string
	URLPrefix string
	URLName string
	Level int
	
	ForumName string //TopicsRow
}

type TopicStore interface {
	Get(id int) (*Topic, error)
	GetUnsafe(id int) (*Topic, error)
	CascadeGet(id int) (*Topic, error)
	Add(item *Topic) error
	AddUnsafe(item *Topic) error
	Remove(id int) error
	RemoveUnsafe(id int) error
	AddLastTopic(item *Topic, fid int) error
	GetLength() int
	GetCapacity() int
}

type StaticTopicStore struct {
	items map[int]*Topic
	length int
	capacity int
	mu sync.RWMutex
}

func NewStaticTopicStore(capacity int) *StaticTopicStore {
	return &StaticTopicStore{items:make(map[int]*Topic),capacity:capacity}
}

func (sts *StaticTopicStore) Get(id int) (*Topic, error) {
	sts.mu.RLock()
	item, ok := sts.items[id]
	sts.mu.RUnlock()
	if ok {
		return item, nil
	}
	return item, sql.ErrNoRows
}

func (sts *StaticTopicStore) GetUnsafe(id int) (*Topic, error) {
	item, ok := sts.items[id]
	if ok {
		return item, nil
	}
	return item, sql.ErrNoRows
}

func (sts *StaticTopicStore) CascadeGet(id int) (*Topic, error) {
	sts.mu.RLock()
	topic, ok := sts.items[id]
	sts.mu.RUnlock()
	if ok {
		return topic, nil
	}
	
	topic = &Topic{ID:id}
	err := get_topic_stmt.QueryRow(id).Scan(&topic.Title, &topic.Content, &topic.CreatedBy, &topic.CreatedAt, &topic.Is_Closed, &topic.Sticky, &topic.ParentID, &topic.IpAddress, &topic.PostCount, &topic.LikeCount)
	if err == nil {
		sts.Add(topic)
	}
	return topic, err
}

func (sts *StaticTopicStore) Add(item *Topic) error {
	if sts.length >= sts.capacity {
		return ErrStoreCapacityOverflow
	}
	sts.mu.Lock()
	sts.items[item.ID] = item
	sts.mu.Unlock()
	sts.length++
	return nil
}

func (sts *StaticTopicStore) AddUnsafe(item *Topic) error {
	if sts.length >= sts.capacity {
		return ErrStoreCapacityOverflow
	}
	sts.items[item.ID] = item
	sts.length++
	return nil
}

func (sts *StaticTopicStore) Remove(id int) error {
	sts.mu.Lock()
	delete(sts.items,id)
	sts.mu.Unlock()
	sts.length--
	return nil
}

func (sts *StaticTopicStore) RemoveUnsafe(id int) error {
	delete(sts.items,id)
	sts.length--
	return nil
}

func (sts *StaticTopicStore) AddLastTopic(item *Topic, fid int) error {
	// Coming Soon...
	return nil
}

func (sts *StaticTopicStore) GetLength() int {
	return sts.length
}

func (sts *StaticTopicStore) SetCapacity(capacity int) {
	sts.capacity = capacity
}

func (sts *StaticTopicStore) GetCapacity() int {
	return sts.capacity
}

//type DynamicTopicStore struct {
//	items_expiries list.List
//	items map[int]*Topic
//}

type SqlTopicStore struct {
}

func NewSqlTopicStore() *SqlTopicStore {
	return &SqlTopicStore{}
}

func (sus *SqlTopicStore) Get(id int) (*Topic, error) {
	topic := Topic{ID:id}
	err := get_topic_stmt.QueryRow(id).Scan(&topic.Title, &topic.Content, &topic.CreatedBy, &topic.CreatedAt, &topic.Is_Closed, &topic.Sticky, &topic.ParentID, &topic.IpAddress, &topic.PostCount, &topic.LikeCount)
	return &topic, err
}

func (sus *SqlTopicStore) GetUnsafe(id int) (*Topic, error) {
	topic := Topic{ID:id}
	err := get_topic_stmt.QueryRow(id).Scan(&topic.Title, &topic.Content, &topic.CreatedBy, &topic.CreatedAt, &topic.Is_Closed, &topic.Sticky, &topic.ParentID, &topic.IpAddress, &topic.PostCount, &topic.LikeCount)
	return &topic, err
}

func (sus *SqlTopicStore) CascadeGet(id int) (*Topic, error) {
	topic := Topic{ID:id}
	err := get_topic_stmt.QueryRow(id).Scan(&topic.Title, &topic.Content, &topic.CreatedBy, &topic.CreatedAt, &topic.Is_Closed, &topic.Sticky, &topic.ParentID, &topic.IpAddress, &topic.PostCount, &topic.LikeCount)
	return &topic, err
}

// Placeholder methods, the actual queries are done elsewhere
func (sus *SqlTopicStore) Add(item *Topic) error {
	return nil
}
func (sus *SqlTopicStore) AddUnsafe(item *Topic) error {
	return nil
}
func (sus *SqlTopicStore) Remove(id int) error {
	return nil
}
func (sus *SqlTopicStore) RemoveUnsafe(id int) error {
	return nil
}
func (sts *SqlTopicStore) AddLastTopic(item *Topic, fid int) error {
	// Coming Soon...
	return nil
}
func (sts *SqlTopicStore) GetCapacity() int {
	return 0
}

func (sus *SqlTopicStore) GetLength() int {
	// Return the total number of topics on the forums
	return 0
}

func get_topicuser(tid int) (TopicUser,error) {
	if cache_topicuser != CACHE_SQL {
		topic, err := topics.Get(tid)
		if err == nil {
			user, err := users.CascadeGet(topic.CreatedBy)
			if err != nil {
				return TopicUser{ID:tid}, err
			}
			
			// We might be better off just passing seperate topic and user structs to the caller?
			return copy_topic_to_topicuser(topic, user), nil
		} else if users.GetLength() < users.GetCapacity() {
			topic, err = topics.CascadeGet(tid)
			if err != nil {
				return TopicUser{ID:tid}, err
			}
			user, err := users.CascadeGet(topic.CreatedBy)
			if err != nil {
				return TopicUser{ID:tid}, err
			}
			tu := copy_topic_to_topicuser(topic, user)
			//fmt.Printf("%+v\n", topic)
			//fmt.Println("")
			//fmt.Printf("%+v\n", user)
			//fmt.Println("")
			//fmt.Printf("%+v\n", tu)
			//fmt.Println("")
			return tu, nil
		}
	}
	
	tu := TopicUser{ID:tid}
	err := get_topic_user_stmt.QueryRow(tid).Scan(&tu.Title, &tu.Content, &tu.CreatedBy, &tu.CreatedAt, &tu.Is_Closed, &tu.Sticky, &tu.ParentID, &tu.IpAddress, &tu.PostCount, &tu.LikeCount, &tu.CreatedByName, &tu.Avatar, &tu.Group, &tu.URLPrefix, &tu.URLName, &tu.Level)
	
	the_topic := Topic{ID:tu.ID, Title:tu.Title, Content:tu.Content, CreatedBy:tu.CreatedBy, Is_Closed:tu.Is_Closed, Sticky:tu.Sticky, CreatedAt:tu.CreatedAt, LastReplyAt:tu.LastReplyAt, ParentID:tu.ParentID, IpAddress:tu.IpAddress, PostCount:tu.PostCount, LikeCount:tu.LikeCount}
	//fmt.Printf("%+v\n", the_topic)
	topics.Add(&the_topic)
	return tu, err
}

func copy_topic_to_topicuser(topic *Topic, user *User) (tu TopicUser) {
	tu.CreatedByName = user.Name
	tu.Group = user.Group
	tu.Avatar = user.Avatar
	tu.URLPrefix = user.URLPrefix
	tu.URLName = user.URLName
	tu.Level = user.Level
	
	tu.ID = topic.ID
	tu.Title = topic.Title
	tu.Content = topic.Content
	tu.CreatedBy = topic.CreatedBy
	tu.Is_Closed = topic.Is_Closed
	tu.Sticky = topic.Sticky
	tu.CreatedAt = topic.CreatedAt
	tu.LastReplyAt = topic.LastReplyAt
	tu.ParentID = topic.ParentID
	tu.IpAddress = topic.IpAddress
	tu.PostCount = topic.PostCount
	tu.LikeCount = topic.LikeCount
	return tu
}