package routes

import (
	"log"
	"net/http"
	"strconv"

	"../common"
)

func TopicList(w http.ResponseWriter, r *http.Request, user common.User) common.RouteError {
	headerVars, ferr := common.UserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	headerVars.Zone = "topics"
	headerVars.MetaDesc = headerVars.Settings["meta_desc"].(string)

	group, err := common.Groups.Get(user.Group)
	if err != nil {
		log.Printf("Group #%d doesn't exist despite being used by common.User #%d", user.Group, user.ID)
		return common.LocalError("Something weird happened", w, r, user)
	}

	// Get the current page
	page, _ := strconv.Atoi(r.FormValue("page"))

	// TODO: Pass a struct back rather than passing back so many variables
	var topicList []*common.TopicsRow
	var forumList []common.Forum
	var pageList []int
	var lastPage int
	if user.IsSuperAdmin {
		topicList, forumList, pageList, page, lastPage, err = common.TopicList.GetList(page)
	} else {
		topicList, forumList, pageList, page, lastPage, err = common.TopicList.GetListByGroup(group, page)
	}
	if err != nil {
		return common.InternalError(err, w, r)
	}

	// ! Need an inline error not a page level error
	//log.Printf("topicList: %+v\n", topicList)
	//log.Printf("forumList: %+v\n", forumList)
	if len(topicList) == 0 {
		return common.NotFound(w, r)
	}

	pi := common.TopicsPage{common.GetTitlePhrase("topics"), user, headerVars, topicList, forumList, common.Config.DefaultForum, pageList, page, lastPage}
	if common.PreRenderHooks["pre_render_topic_list"] != nil {
		if common.RunPreRenderHook("pre_render_topic_list", w, r, &user, &pi) {
			return nil
		}
	}
	err = common.RunThemeTemplate(headerVars.Theme.Name, "topics", pi, w)
	if err != nil {
		return common.InternalError(err, w, r)
	}
	return nil
}
