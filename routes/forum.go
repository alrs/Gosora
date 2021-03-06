package routes

import (
	"database/sql"
	"net/http"
	"strconv"

	c "github.com/Azareal/Gosora/common"
	"github.com/Azareal/Gosora/common/counters"
	p "github.com/Azareal/Gosora/common/phrases"
	qgen "github.com/Azareal/Gosora/query_gen"
)

type ForumStmts struct {
	getTopics *sql.Stmt
}

var forumStmts ForumStmts

// TODO: Move these DbInits into *Forum as Topics()
func init() {
	c.DbInits.Add(func(acc *qgen.Accumulator) error {
		forumStmts = ForumStmts{
			getTopics: acc.Select("topics").Columns("tid, title, content, createdBy, is_closed, sticky, createdAt, lastReplyAt, lastReplyBy, lastReplyID, parentID, views, postCount, likeCount").Where("parentID = ?").Orderby("sticky DESC, lastReplyAt DESC, createdBy DESC").Limit("?,?").Prepare(),
		}
		return acc.FirstError()
	})
}

// TODO: Retire this in favour of an alias for /topics/?
func ViewForum(w http.ResponseWriter, r *http.Request, user c.User, header *c.Header, sfid string) c.RouteError {
	page, _ := strconv.Atoi(r.FormValue("page"))
	_, fid, err := ParseSEOURL(sfid)
	if err != nil {
		return c.SimpleError(p.GetErrorPhrase("url_id_must_be_integer"), w, r, header)
	}

	ferr := c.ForumUserCheck(header, w, r, &user, fid)
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ViewTopic {
		return c.NoPermissions(w, r, user)
	}
	header.Path = "/forums/"

	// TODO: Fix this double-check
	forum, err := c.Forums.Get(fid)
	if err == sql.ErrNoRows {
		return c.NotFound(w, r, header)
	} else if err != nil {
		return c.InternalError(err, w, r)
	}
	header.Title = forum.Name
	header.OGDesc = forum.Desc

	// TODO: Does forum.TopicCount take the deleted items into consideration for guests? We don't have soft-delete yet, only hard-delete
	offset, page, lastPage := c.PageOffset(forum.TopicCount, page, c.Config.ItemsPerPage)

	// TODO: Move this to *Forum
	rows, err := forumStmts.getTopics.Query(fid, offset, c.Config.ItemsPerPage)
	if err != nil {
		return c.InternalError(err, w, r)
	}
	defer rows.Close()

	// TODO: Use something other than TopicsRow as we don't need to store the forum name and link on each and every topic item?
	var topicList []*c.TopicsRow
	reqUserList := make(map[int]bool)
	for rows.Next() {
		t := c.TopicsRow{ID: 0}
		err := rows.Scan(&t.ID, &t.Title, &t.Content, &t.CreatedBy, &t.IsClosed, &t.Sticky, &t.CreatedAt, &t.LastReplyAt, &t.LastReplyBy, &t.LastReplyID, &t.ParentID, &t.ViewCount, &t.PostCount, &t.LikeCount)
		if err != nil {
			return c.InternalError(err, w, r)
		}

		t.Link = c.BuildTopicURL(c.NameToSlug(t.Title), t.ID)
		// TODO: Create a specialised function with a bit less overhead for getting the last page for a post count
		_, _, lastPage := c.PageOffset(t.PostCount, 1, c.Config.ItemsPerPage)
		t.LastPage = lastPage

		header.Hooks.VhookNoRet("forum_trow_assign", &t, &forum)
		topicList = append(topicList, &t)
		reqUserList[t.CreatedBy] = true
		reqUserList[t.LastReplyBy] = true
	}
	err = rows.Err()
	if err != nil {
		return c.InternalError(err, w, r)
	}

	// Convert the user ID map to a slice, then bulk load the users
	idSlice := make([]int, len(reqUserList))
	var i int
	for userID := range reqUserList {
		idSlice[i] = userID
		i++
	}

	// TODO: What if a user is deleted via the Control Panel?
	userList, err := c.Users.BulkGetMap(idSlice)
	if err != nil {
		return c.InternalError(err, w, r)
	}

	// Second pass to the add the user data
	// TODO: Use a pointer to TopicsRow instead of TopicsRow itself?
	for _, t := range topicList {
		t.Creator = userList[t.CreatedBy]
		t.LastUser = userList[t.LastReplyBy]
	}
	header.Zone = "view_forum"
	header.ZoneID = forum.ID

	// TODO: Reduce the amount of boilerplate here
	if r.FormValue("js") == "1" {
		outBytes, err := wsTopicList(topicList, lastPage).MarshalJSON()
		if err != nil {
			return c.InternalError(err, w, r)
		}
		w.Write(outBytes)
		return nil
	}

	pageList := c.Paginate(page, lastPage, 5)
	pi := c.ForumPage{header, topicList, forum, c.Paginator{pageList, page, lastPage}}
	tmpl := forum.Tmpl
	if tmpl == "" {
		ferr = renderTemplate("forum", w, r, header, pi)
	} else {
		tmpl = "forum_" + tmpl
		err = renderTemplate3(tmpl, tmpl, w, r, header, pi)
		if err != nil {
			ferr = renderTemplate("forum", w, r, header, pi)
		}
	}
	counters.ForumViewCounter.Bump(forum.ID)
	return ferr
}
