package routes

import (
	"log"
	"net/http"

	"../common"
)

func ForumList(w http.ResponseWriter, r *http.Request, user common.User) common.RouteError {
	headerVars, ferr := common.UserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	headerVars.Zone = "forums"
	headerVars.MetaDesc = headerVars.Settings["meta_desc"].(string)

	var err error
	var forumList []common.Forum
	var canSee []int
	if user.IsSuperAdmin {
		canSee, err = common.Forums.GetAllVisibleIDs()
		if err != nil {
			return common.InternalError(err, w, r)
		}
	} else {
		group, err := common.Groups.Get(user.Group)
		if err != nil {
			log.Printf("Group #%d doesn't exist despite being used by common.User #%d", user.Group, user.ID)
			return common.LocalError("Something weird happened", w, r, user)
		}
		canSee = group.CanSee
	}

	for _, fid := range canSee {
		// Avoid data races by copying the struct into something we can freely mold without worrying about breaking something somewhere else
		var forum = common.Forums.DirtyGet(fid).Copy()
		if forum.ParentID == 0 && forum.Name != "" && forum.Active {
			if forum.LastTopicID != 0 {
				if forum.LastTopic.ID != 0 && forum.LastReplyer.ID != 0 {
					forum.LastTopicTime = common.RelativeTime(forum.LastTopic.LastReplyAt)
				} else {
					forum.LastTopicTime = ""
				}
			} else {
				forum.LastTopicTime = ""
			}
			common.RunHook("forums_frow_assign", &forum)
			forumList = append(forumList, forum)
		}
	}

	pi := common.ForumsPage{common.GetTitlePhrase("forums"), user, headerVars, forumList}
	if common.RunPreRenderHook("pre_render_forum_list", w, r, &user, &pi) {
		return nil
	}
	err = common.RunThemeTemplate(headerVars.Theme.Name, "forums", pi, w)
	if err != nil {
		return common.InternalError(err, w, r)
	}
	return nil
}