package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"../common"
)

func PollVote(w http.ResponseWriter, r *http.Request, user common.User, sPollID string) common.RouteError {
	pollID, err := strconv.Atoi(sPollID)
	if err != nil {
		return common.PreError("The provided PollID is not a valid number.", w, r)
	}

	poll, err := common.Polls.Get(pollID)
	if err == sql.ErrNoRows {
		return common.PreError("The poll you tried to vote for doesn't exist.", w, r)
	} else if err != nil {
		return common.InternalError(err, w, r)
	}

	var topic *common.Topic
	if poll.ParentTable == "replies" {
		reply, err := common.Rstore.Get(poll.ParentID)
		if err == sql.ErrNoRows {
			return common.PreError("The parent post doesn't exist.", w, r)
		} else if err != nil {
			return common.InternalError(err, w, r)
		}
		topic, err = common.Topics.Get(reply.ParentID)
	} else if poll.ParentTable == "topics" {
		topic, err = common.Topics.Get(poll.ParentID)
	} else {
		return common.InternalError(errors.New("Unknown parentTable for poll"), w, r)
	}

	if err == sql.ErrNoRows {
		return common.PreError("The parent topic doesn't exist.", w, r)
	} else if err != nil {
		return common.InternalError(err, w, r)
	}

	// TODO: Add hooks to make use of headerLite
	_, ferr := common.SimpleForumUserCheck(w, r, &user, topic.ParentID)
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ViewTopic {
		return common.NoPermissions(w, r, user)
	}

	optionIndex, err := strconv.Atoi(r.PostFormValue("poll_option_input"))
	if err != nil {
		return common.LocalError("Malformed input", w, r, user)
	}

	err = poll.CastVote(optionIndex, user.ID, user.LastIP)
	if err != nil {
		return common.InternalError(err, w, r)
	}

	http.Redirect(w, r, "/topic/"+strconv.Itoa(topic.ID), http.StatusSeeOther)
	return nil
}
