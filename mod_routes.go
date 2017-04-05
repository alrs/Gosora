package main

import "log"
import "fmt"
import "strings"
import "strconv"
import "net"
import "net/http"
import "html"
import "encoding/json"
import "database/sql"
import _ "github.com/go-sql-driver/mysql"

func route_edit_topic(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		PreError("Bad Form",w,r)
		return          
	}
	is_js := r.PostFormValue("js")
	if is_js == "" {
		is_js = "0"
	}
	
	var tid int
	var fid int
	tid, err = strconv.Atoi(r.URL.Path[len("/topic/edit/submit/"):])
	if err != nil {
		PreErrorJSQ("The provided TopicID is not a valid number.",w,r,is_js)
		return
	}
	
	var old_is_closed bool
	err = db.QueryRow("select parentID, is_closed from topics where tid = ?", tid).Scan(&fid,&old_is_closed)
	if err == sql.ErrNoRows {
		PreErrorJSQ("The topic you tried to edit doesn't exist.",w,r,is_js)
		return
	} else if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	user, ok := SimpleForumSessionCheck(w,r,fid)
	if !ok {
		return
	}
	if !user.Perms.ViewTopic || !user.Perms.EditTopic {
		NoPermissionsJSQ(w,r,user,is_js)
		return
	}
	
	topic_name := r.PostFormValue("topic_name")
	topic_status := r.PostFormValue("topic_status")
	is_closed := (topic_status == "closed")
	
	topic_content := html.EscapeString(r.PostFormValue("topic_content"))
	_, err = edit_topic_stmt.Exec(topic_name, preparse_message(topic_content), parse_message(html.EscapeString(preparse_message(topic_content))), is_closed, tid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	ipaddress, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		LocalError("Bad IP",w,r,user)
		return
	}
	
	if old_is_closed != is_closed {
		var action string
		if is_closed {
			action = "lock"
		} else {
			action = "unlock"
		}
		
		err = addModLog(action,tid,"topic",ipaddress,user.ID)
		if err != nil {
			InternalError(err,w,r)
			return
		}
		_, err = create_action_reply_stmt.Exec(tid,action,ipaddress,user.ID)
		if err != nil {
			InternalError(err,w,r)
			return
		}
		
		_, err = add_replies_to_topic_stmt.Exec(1, tid)
		if err != nil {
			InternalError(err,w,r)
			return
		}
		_, err = update_forum_cache_stmt.Exec(topic_name, tid, user.Name, user.ID, 1)
		if err != nil {
			InternalError(err,w,r)
			return
		}
	}
	
	err = topics.Load(tid)
	if err != nil {
		LocalErrorJSQ("This topic no longer exists!",w,r,user,is_js)
		return
	}
	
	if is_js == "0" {
		http.Redirect(w,r,"/topic/" + strconv.Itoa(tid),http.StatusSeeOther)
	} else {
		fmt.Fprintf(w,`{"success":"1"}`)
	}
}

func route_delete_topic(w http.ResponseWriter, r *http.Request) {
	tid, err := strconv.Atoi(r.URL.Path[len("/topic/delete/submit/"):])
	if err != nil {
		PreError("The provided TopicID is not a valid number.",w,r)
		return
	}
	
	var content string
	var createdBy int
	var fid int
	err = db.QueryRow("select content, createdBy, parentID from topics where tid = ?", tid).Scan(&content, &createdBy, &fid)
	if err == sql.ErrNoRows {
		PreError("The topic you tried to delete doesn't exist.",w,r)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	user, ok := SimpleForumSessionCheck(w,r,fid)
	if !ok {
		return
	}
	if !user.Perms.ViewTopic || !user.Perms.DeleteTopic {
		NoPermissions(w,r,user)
		return
	}
	
	_, err = delete_topic_stmt.Exec(tid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	ipaddress, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		LocalError("Bad IP",w,r,user)
		return
	}
	err = addModLog("delete",tid,"topic",ipaddress,user.ID)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	/*_, err = create_action_reply_stmt.Exec(tid,"delete",ipaddress,user.ID)
	if err != nil {
		InternalError(err,w,r)
		return
	}*/
	
	log.Print("The topic '" + strconv.Itoa(tid) + "' was deleted by User ID #" + strconv.Itoa(user.ID) + ".")
	http.Redirect(w,r,"/",http.StatusSeeOther)
	
	wcount := word_count(content)
	err = decrease_post_user_stats(wcount,createdBy,true,user)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	_, err = remove_topics_from_forum_stmt.Exec(1, fid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	forums[fid].TopicCount -= 1
	topics.Remove(tid)
}

func route_stick_topic(w http.ResponseWriter, r *http.Request) {
	tid, err := strconv.Atoi(r.URL.Path[len("/topic/stick/submit/"):])
	if err != nil {
		PreError("The provided TopicID is not a valid number.",w,r)
		return
	}
	
	topic, err := topics.CascadeGet(tid)
	if err == sql.ErrNoRows {
		PreError("The topic you tried to pin doesn't exist.",w,r)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	user, ok := SimpleForumSessionCheck(w,r,topic.ParentID)
	if !ok {
		return
	}
	if !user.Perms.ViewTopic || !user.Perms.PinTopic {
		NoPermissions(w,r,user)
		return
	}
	
	_, err = stick_topic_stmt.Exec(tid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	//topic.Sticky = true
	err = topics.Load(tid)
	if err != nil {
		LocalError("This topic doesn't exist!",w,r,user)
		return
	}
	http.Redirect(w,r,"/topic/" + strconv.Itoa(tid),http.StatusSeeOther)
}

func route_unstick_topic(w http.ResponseWriter, r *http.Request) {
	tid, err := strconv.Atoi(r.URL.Path[len("/topic/unstick/submit/"):])
	if err != nil {
		PreError("The provided TopicID is not a valid number.",w,r)
		return
	}
	
	topic, err := topics.CascadeGet(tid)
	if err == sql.ErrNoRows {
		PreError("The topic you tried to unpin doesn't exist.",w,r)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	user, ok := SimpleForumSessionCheck(w,r,topic.ParentID)
	if !ok {
		return
	}
	if !user.Perms.ViewTopic || !user.Perms.PinTopic {
		NoPermissions(w,r,user)
		return
	}
	
	_, err = unstick_topic_stmt.Exec(tid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	//topic.Sticky = false
	err = topics.Load(tid)
	if err != nil {
		LocalError("This topic doesn't exist!",w,r,user)
		return
	}
	http.Redirect(w,r,"/topic/" + strconv.Itoa(tid),http.StatusSeeOther)
}

func route_reply_edit_submit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		PreError("Bad Form",w,r)
		return          
	}
	is_js := r.PostFormValue("js")
	if is_js == "" {
		is_js = "0"
	}
	
	rid, err := strconv.Atoi(r.URL.Path[len("/reply/edit/submit/"):])
	if err != nil {
		PreErrorJSQ("The provided Reply ID is not a valid number.",w,r,is_js)
		return
	}
	
	content := html.EscapeString(preparse_message(r.PostFormValue("edit_item")))
	_, err = edit_reply_stmt.Exec(content, parse_message(content), rid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	// Get the Reply ID..
	var tid int
	err = db.QueryRow("select tid from replies where rid = ?", rid).Scan(&tid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	var fid int
	err = db.QueryRow("select parentID from topics where tid = ?", tid).Scan(&fid)
	if err == sql.ErrNoRows {
		PreErrorJSQ("The parent topic doesn't exist.",w,r,is_js)
		return
	} else if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	user, ok := SimpleForumSessionCheck(w,r,fid)
	if !ok {
		return
	}
	if !user.Perms.ViewTopic || !user.Perms.EditReply {
		NoPermissionsJSQ(w,r,user,is_js)
		return
	}
	
	if is_js == "0" {
		http.Redirect(w,r, "/topic/" + strconv.Itoa(tid) + "#reply-" + strconv.Itoa(rid), http.StatusSeeOther)
	} else {
		fmt.Fprintf(w,`{"success":"1"}`)
	}
}

func route_reply_delete_submit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		PreError("Bad Form",w,r)
		return          
	}
	is_js := r.PostFormValue("is_js")
	if is_js == "" {
		is_js = "0"
	}
	
	rid, err := strconv.Atoi(r.URL.Path[len("/reply/delete/submit/"):])
	if err != nil {
		PreErrorJSQ("The provided Reply ID is not a valid number.",w,r,is_js)
		return
	}
	
	var tid int
	var content string
	var createdBy int
	err = db.QueryRow("select tid, content, createdBy from replies where rid = ?", rid).Scan(&tid, &content, &createdBy)
	if err == sql.ErrNoRows {
		PreErrorJSQ("The reply you tried to delete doesn't exist.",w,r,is_js)
		return
	} else if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	var fid int
	err = db.QueryRow("select parentID from topics where tid = ?", tid).Scan(&fid)
	if err == sql.ErrNoRows {
		PreErrorJSQ("The parent topic doesn't exist.",w,r,is_js)
		return
	} else if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	user, ok := SimpleForumSessionCheck(w,r,fid)
	if !ok {
		return
	}
	if !user.Perms.ViewTopic || !user.Perms.DeleteReply {
		NoPermissionsJSQ(w,r,user,is_js)
		return
	}
	
	_, err = delete_reply_stmt.Exec(rid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	log.Print("The reply '" + strconv.Itoa(rid) + "' was deleted by User ID #" + strconv.Itoa(user.ID) + ".")
	if is_js == "0" {
		//http.Redirect(w,r, "/topic/" + strconv.Itoa(tid), http.StatusSeeOther)
	} else {
		fmt.Fprintf(w,`{"success":"1"}`)
	}
	
	wcount := word_count(content)
	err = decrease_post_user_stats(wcount, createdBy, false, user)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	_, err = remove_replies_from_topic_stmt.Exec(1,tid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
	}
	
	err = topics.Load(tid)
	if err != nil {
		LocalError("This topic no longer exists!",w,r,user)
		return
	}
}

func route_profile_reply_edit_submit(w http.ResponseWriter, r *http.Request) {
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	
	err := r.ParseForm()
	if err != nil {
		LocalError("Bad Form",w,r,user)
		return          
	}
	is_js := r.PostFormValue("js")
	if is_js == "" {
		is_js = "0"
	}
	
	rid, err := strconv.Atoi(r.URL.Path[len("/profile/reply/edit/submit/"):])
	if err != nil {
		LocalErrorJSQ("The provided Reply ID is not a valid number.",w,r,user,is_js)
		return
	}
	
	// Get the Reply ID..
	var uid int
	err = db.QueryRow("select uid from users_replies where rid = ?", rid).Scan(&uid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	if user.ID != uid && !user.Perms.EditReply {
		NoPermissionsJSQ(w,r,user,is_js)
		return
	}
	
	content := html.EscapeString(preparse_message(r.PostFormValue("edit_item")))
	_, err = edit_profile_reply_stmt.Exec(content, parse_message(content), rid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	if is_js == "0" {
		http.Redirect(w,r, "/user/" + strconv.Itoa(uid) + "#reply-" + strconv.Itoa(rid), http.StatusSeeOther)
	} else {
		fmt.Fprintf(w,`{"success":"1"}`)
	}
}

func route_profile_reply_delete_submit(w http.ResponseWriter, r *http.Request) {
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	
	err := r.ParseForm()
	if err != nil {
		LocalError("Bad Form",w,r,user)
		return          
	}
	is_js := r.PostFormValue("is_js")
	if is_js == "" {
		is_js = "0"
	}
	
	rid, err := strconv.Atoi(r.URL.Path[len("/profile/reply/delete/submit/"):])
	if err != nil {
		LocalErrorJSQ("The provided Reply ID is not a valid number.",w,r,user,is_js)
		return
	}
	
	var uid int
	err = db.QueryRow("select uid from users_replies where rid = ?", rid).Scan(&uid)
	if err == sql.ErrNoRows {
		LocalErrorJSQ("The reply you tried to delete doesn't exist.",w,r,user,is_js)
		return
	} else if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	if user.ID != uid && !user.Perms.DeleteReply {
		NoPermissionsJSQ(w,r,user,is_js)
		return
	}
	
	_, err = delete_profile_reply_stmt.Exec(rid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	log.Print("The reply '" + strconv.Itoa(rid) + "' was deleted by User ID #" + strconv.Itoa(user.ID) + ".")
	
	if is_js == "0" {
		//http.Redirect(w,r, "/user/" + strconv.Itoa(uid), http.StatusSeeOther)
	} else {
		fmt.Fprintf(w,`{"success":"1"}`)
	}
}

func route_ban(w http.ResponseWriter, r *http.Request) {
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Perms.BanUsers {
		NoPermissions(w,r,user)
		return
	}
	
	uid, err := strconv.Atoi(r.URL.Path[len("/users/ban/"):])
	if err != nil {
		LocalError("The provided User ID is not a valid number.",w,r,user)
		return
	}
	
	var uname string
	err = db.QueryRow("select name from users where uid = ?", uid).Scan(&uname)
	if err == sql.ErrNoRows {
		LocalError("The user you're trying to ban no longer exists.",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	confirm_msg := "Are you sure you want to ban '" + uname + "'?"
	yousure := AreYouSure{"/users/ban/submit/" + strconv.Itoa(uid),confirm_msg}
	
	pi := Page{"Ban User",user,noticeList,tList,yousure}
	templates.ExecuteTemplate(w,"areyousure.html",pi)
}

func route_ban_submit(w http.ResponseWriter, r *http.Request) {
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Perms.BanUsers {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	uid, err := strconv.Atoi(r.URL.Path[len("/users/ban/submit/"):])
	if err != nil {
		LocalError("The provided User ID is not a valid number.",w,r,user)
		return
	}
	
	var group int
	var is_super_admin bool
	err = db.QueryRow("select `group`,`is_super_admin` from `users` where `uid` = ?", uid).Scan(&group, &is_super_admin)
	if err == sql.ErrNoRows {
		LocalError("The user you're trying to ban no longer exists.",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	if is_super_admin || groups[group].Is_Admin || groups[group].Is_Mod {
		LocalError("You may not ban another staff member.",w,r,user)
		return
	}
	if uid == user.ID {
		LocalError("You may not ban yourself.",w,r,user)
		return
	}
	if uid == -2 {
		LocalError("You may not ban me. Fine, I will offer up some guidance unto thee. Come to my lair, young one. /arcane-tower/",w,r,user)
		return
	}
	
	if groups[group].Is_Banned {
		LocalError("The user you're trying to unban is already banned.",w,r,user)
		return
	}
	
	_, err = change_group_stmt.Exec(4, uid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	err = users.Load(uid)
	if err != nil {
		LocalError("This user no longer exists!",w,r,user)
		return
	}
	http.Redirect(w,r,"/users/" + strconv.Itoa(uid),http.StatusSeeOther)
}

func route_unban(w http.ResponseWriter, r *http.Request) {
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Perms.BanUsers {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	uid, err := strconv.Atoi(r.URL.Path[len("/users/unban/"):])
	if err != nil {
		LocalError("The provided User ID is not a valid number.",w,r,user)
		return
	}
	
	var uname string
	var group int
	err = db.QueryRow("select `name`, `group` from users where `uid` = ?", uid).Scan(&uname, &group)
	if err == sql.ErrNoRows {
		LocalError("The user you're trying to unban no longer exists.",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	if !groups[group].Is_Banned {
		LocalError("The user you're trying to unban isn't banned.",w,r,user)
		return
	}
	
	_, err = change_group_stmt.Exec(default_group, uid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	err = users.Load(uid)
	if err != nil {
		LocalError("This user no longer exists!",w,r,user)
		return
	}
	http.Redirect(w,r,"/users/" + strconv.Itoa(uid),http.StatusSeeOther)
}

func route_activate(w http.ResponseWriter, r *http.Request) {
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Perms.ActivateUsers {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	uid, err := strconv.Atoi(r.URL.Path[len("/users/activate/"):])
	if err != nil {
		LocalError("The provided User ID is not a valid number.",w,r,user)
		return
	}
	
	var uname string
	var active bool
	err = db.QueryRow("select `name`,`active` from users where `uid` = ?", uid).Scan(&uname, &active)
	if err == sql.ErrNoRows {
		LocalError("The account you're trying to activate no longer exists.",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	if active {
		LocalError("The account you're trying to activate has already been activated.",w,r,user)
		return
	}
	_, err = activate_user_stmt.Exec(uid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	_, err = change_group_stmt.Exec(default_group, uid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	err = users.Load(uid)
	if err != nil {
		LocalError("This user no longer exists!",w,r,user)
		return
	}
	http.Redirect(w,r,"/users/" + strconv.Itoa(uid),http.StatusSeeOther)
}

/* Control Panel*/
func route_panel(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod {
		NoPermissions(w,r,user)
		return
	}
	pi := Page{"Control Panel Dashboard",user,noticeList,tList,nil}
	templates.ExecuteTemplate(w,"panel-dashboard.html",pi)
}

func route_panel_forums(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManageForums {
		NoPermissions(w,r,user)
		return
	}
	
	var forumList []interface{}
	for _, forum := range forums {
		if forum.Name != "" {
			fadmin := ForumAdmin{forum.ID,forum.Name,forum.Active,forum.Preset,forum.TopicCount,preset_to_lang(forum.Preset),preset_to_emoji(forum.Preset)}
			forumList = append(forumList,fadmin)
		}
	}
	pi := Page{"Forum Manager",user,noticeList,forumList,nil}
	templates.ExecuteTemplate(w,"panel-forums.html",pi)
}

func route_panel_forums_create_submit(w http.ResponseWriter, r *http.Request){
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManageForums {
		NoPermissions(w,r,user)
		return
	}
	
	err := r.ParseForm()
	if err != nil {
		LocalError("Bad Form",w,r,user)
		return          
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	var active bool
	fname := r.PostFormValue("forum-name")
	fpreset := strip_invalid_preset(r.PostFormValue("forum-preset"))
	factive := r.PostFormValue("forum-name")
	if factive == "on" || factive == "1" {
		active = true
	} else {
		active = false
	}
	
	fid, err := create_forum(fname,active,fpreset)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	permmap_to_query(preset_to_permmap(fpreset),fid)
	http.Redirect(w,r,"/panel/forums/",http.StatusSeeOther)
}

func route_panel_forums_delete(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManageForums {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	fid, err := strconv.Atoi(r.URL.Path[len("/panel/forums/delete/"):])
	if err != nil {
		LocalError("The provided Forum ID is not a valid number.",w,r,user)
		return
	}
	
	if !forum_exists(fid) {
		LocalError("The forum you're trying to delete doesn't exist.",w,r,user)
		return
	}
	
	confirm_msg := "Are you sure you want to delete the '" + forums[fid].Name + "' forum?"
	yousure := AreYouSure{"/panel/forums/delete/submit/" + strconv.Itoa(fid),confirm_msg}
	
	pi := Page{"Delete Forum",user,noticeList,tList,yousure}
	templates.ExecuteTemplate(w,"areyousure.html",pi)
}

func route_panel_forums_delete_submit(w http.ResponseWriter, r *http.Request) {
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManageForums {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	fid, err := strconv.Atoi(r.URL.Path[len("/panel/forums/delete/submit/"):])
	if err != nil {
		LocalError("The provided Forum ID is not a valid number.",w,r,user)
		return
	}
	if !forum_exists(fid) {
		LocalError("The forum you're trying to delete doesn't exist.",w,r,user)
		return
	}
	
	err = delete_forum(fid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	http.Redirect(w,r,"/panel/forums/",http.StatusSeeOther)
}

func route_panel_forums_edit(w http.ResponseWriter, r *http.Request) {
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManageForums {
		NoPermissions(w,r,user)
		return
	}
	
	fid, err := strconv.Atoi(r.URL.Path[len("/panel/forums/edit/"):])
	if err != nil {
		LocalError("The provided Forum ID is not a valid number.",w,r,user)
		return
	}
	if !forum_exists(fid) {
		LocalError("The forum you're trying to edit doesn't exist.",w,r,user)
		return
	}
	
	pi := Page{"Forum Editor",user,noticeList,tList,nil}
	templates.ExecuteTemplate(w,"panel-forum-edit.html",pi)
}

func route_panel_forums_edit_submit(w http.ResponseWriter, r *http.Request) {
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManageForums {
		NoPermissions(w,r,user)
		return
	}
	
	err := r.ParseForm()
	if err != nil {
		LocalError("Bad Form",w,r,user)
		return          
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	is_js := r.PostFormValue("js")
	if is_js == "" {
		is_js = "0"
	}
	
	fid, err := strconv.Atoi(r.URL.Path[len("/panel/forums/edit/submit/"):])
	if err != nil {
		LocalErrorJSQ("The provided Forum ID is not a valid number.",w,r,user,is_js)
		return
	}
	
	forum_name := r.PostFormValue("forum-name")
	forum_preset := strip_invalid_preset(r.PostFormValue("forum-preset"))
	forum_active := r.PostFormValue("forum-active")
    if !forum_exists(fid) {
		LocalErrorJSQ("The forum you're trying to edit doesn't exist.",w,r,user,is_js)
		return
	}
	
	/*if forum_name == "" && forum_active == "" {
		LocalErrorJSQ("You haven't changed anything!",w,r,user,is_js)
		return
	}*/
	
	if forum_name == "" {
		forum_name = forums[fid].Name
	}
	
	var active bool
	if forum_active == "" {
		active = forums[fid].Active
	} else if forum_active == "1" || forum_active == "Show" {
		active = true
	} else {
		active = false
	}
	
	_, err = update_forum_stmt.Exec(forum_name,active,forum_preset,fid)
	if err != nil {
		InternalErrorJSQ(err,w,r,is_js)
		return
	}
	
	if forums[fid].Name != forum_name {
		forums[fid].Name = forum_name
	}
	if forums[fid].Active != active {
		forums[fid].Active = active
	}
	if forums[fid].Preset != forum_preset {
		forums[fid].Preset = forum_preset
	}
	
	permmap_to_query(preset_to_permmap(forum_preset),fid)
	
	if is_js == "0" {
		http.Redirect(w,r,"/panel/forums/",http.StatusSeeOther)
	} else {
		fmt.Fprintf(w,`{"success":"1"}`)
	}
}

func route_panel_settings(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditSettings {
		NoPermissions(w,r,user)
		return
	}
	
	var settingList map[string]interface{} = make(map[string]interface{})
	rows, err := db.Query("select name, content, type from settings")
	if err != nil {
		InternalError(err,w,r)
		return
	}
	defer rows.Close()
	
	var sname string
	var scontent string
	var stype string
	for rows.Next() {
		err := rows.Scan(&sname,&scontent,&stype)
		if err != nil {
			InternalError(err,w,r)
			return
		}
		
		if stype == "list" {
			llist := settingLabels[sname]
			labels := strings.Split(llist,",")
			conv, err := strconv.Atoi(scontent)
			if err != nil {
				LocalError("The setting '" + sname + "' can't be converted to an integer",w,r,user)
				return
			}
			scontent = labels[conv - 1]
		} else if stype == "bool" {
			if scontent == "1" {
				scontent = "Yes"
			} else {
				scontent = "No"
			}
		}
		settingList[sname] = scontent
	}
	err = rows.Err()
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	pi := Page{"Setting Manager",user,noticeList,tList,settingList}
	templates.ExecuteTemplate(w,"panel-settings.html",pi)
}

func route_panel_setting(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditSettings {
		NoPermissions(w,r,user)
		return
	}
	
	setting := Setting{"","","",""}
	setting.Name = r.URL.Path[len("/panel/settings/edit/"):]
	
	err := db.QueryRow("select content, type from settings where name = ?", setting.Name).Scan(&setting.Content,&setting.Type)
	if err == sql.ErrNoRows {
		LocalError("The setting you want to edit doesn't exist.",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	var itemList []interface{}
	if setting.Type == "list" {
		llist, ok := settingLabels[setting.Name]
		if !ok {
			LocalError("The labels for this setting don't exist",w,r,user)
			return
		}
		
		conv, err := strconv.Atoi(setting.Content)
		if err != nil {
			LocalError("The value of this setting couldn't be converted to an integer",w,r,user)
			return
		}
		
		labels := strings.Split(llist,",")
		for index, label := range labels {
			itemList = append(itemList, OptionLabel{
				Label: label,
				Value: index + 1,
				Selected: conv == (index + 1),
			})
		}
	}
	
	pi := Page{"Edit Setting",user,noticeList,itemList,setting}
	templates.ExecuteTemplate(w,"panel-setting.html",pi)
}

func route_panel_setting_edit(w http.ResponseWriter, r *http.Request) {
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditSettings {
		NoPermissions(w,r,user)
		return
	}
	
	err := r.ParseForm()
	if err != nil {
		LocalError("Bad Form",w,r,user)
		return          
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	var stype string
	var sconstraints string
	sname := r.URL.Path[len("/panel/settings/edit/submit/"):]
	scontent := r.PostFormValue("setting-value")
	
	err = db.QueryRow("select name, type, constraints from settings where name = ?", sname).Scan(&sname, &stype, &sconstraints)
	if err == sql.ErrNoRows {
		LocalError("The setting you want to edit doesn't exist.",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	if stype == "bool" {
		if scontent == "on" || scontent == "1" {
			scontent = "1"
		} else {
			scontent = "0"
		}
	}
	
	_, err = update_setting_stmt.Exec(scontent,sname)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	errmsg := parseSetting(sname, scontent, stype, sconstraints)
	if errmsg != "" {
		LocalError(errmsg,w,r,user)
		return
	}
	http.Redirect(w,r,"/panel/settings/",http.StatusSeeOther)
}

func route_panel_plugins(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManagePlugins {
		NoPermissions(w,r,user)
		return
	}
	
	var pluginList []interface{}
	for _, plugin := range plugins {
		pluginList = append(pluginList,plugin)
	}
	
	pi := Page{"Plugin Manager",user,noticeList,pluginList,nil}
	templates.ExecuteTemplate(w,"panel-plugins.html",pi)
}

func route_panel_plugins_activate(w http.ResponseWriter, r *http.Request){
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManagePlugins {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	uname := r.URL.Path[len("/panel/plugins/activate/"):]
	plugin, ok := plugins[uname]
	if !ok {
		LocalError("The plugin isn't registered in the system",w,r,user)
		return
	}
	
	var active bool
	err := db.QueryRow("select active from plugins where uname = ?", uname).Scan(&active)
	if err != nil && err != sql.ErrNoRows {
		InternalError(err,w,r)
		return
	}
	
	if plugins[uname].Activate != nil {
		err = plugins[uname].Activate()
		if err != nil {
			LocalError(err.Error(),w,r,user)
			return
		}
	}
	
	has_plugin := err != sql.ErrNoRows
	if has_plugin {
		if active {
			LocalError("The plugin is already active",w,r,user)
			return
		}
		_, err = update_plugin_stmt.Exec(1,uname)
		if err != nil {
			InternalError(err,w,r)
			return
		}
	} else {
		_, err := add_plugin_stmt.Exec(uname,1)
		if err != nil {
			InternalError(err,w,r)
			return
		}
	}
	
	log.Print("Activating plugin '" + plugin.Name + "'")
	plugin.Active = true
	plugins[uname] = plugin
	plugins[uname].Init()
	http.Redirect(w,r,"/panel/plugins/",http.StatusSeeOther)
}

func route_panel_plugins_deactivate(w http.ResponseWriter, r *http.Request){
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManagePlugins {
		NoPermissions(w,r,user)
		return
	}
	
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	uname := r.URL.Path[len("/panel/plugins/deactivate/"):]
	plugin, ok := plugins[uname]
	if !ok {
		LocalError("The plugin isn't registered in the system",w,r,user)
		return
	}
	
	var active bool
	err := db.QueryRow("select active from plugins where uname = ?", uname).Scan(&active)
	if err == sql.ErrNoRows {
		LocalError("The plugin you're trying to deactivate isn't active",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	if !active {
		LocalError("The plugin you're trying to deactivate isn't active",w,r,user)
		return
	}
	_, err = update_plugin_stmt.Exec(0,uname)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	plugin.Active = false
	plugins[uname] = plugin
	plugins[uname].Deactivate()
	
	http.Redirect(w,r,"/panel/plugins/",http.StatusSeeOther)
}

func route_panel_users(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod {
		NoPermissions(w,r,user)
		return
	}
	
	var userList []interface{}
	rows, err := db.Query("select `uid`,`name`,`group`,`active`,`is_super_admin`,`avatar` from users")
	if err != nil {
		InternalError(err,w,r)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		puser := User{ID: 0,}
		err := rows.Scan(&puser.ID, &puser.Name, &puser.Group, &puser.Active, &puser.Is_Super_Admin, &puser.Avatar)
		if err != nil {
			InternalError(err,w,r)
			return
		}
		
		puser.Is_Admin = puser.Is_Super_Admin || groups[puser.Group].Is_Admin
		puser.Is_Super_Mod = puser.Is_Admin || groups[puser.Group].Is_Mod
		puser.Is_Mod = puser.Is_Super_Mod
		puser.Is_Banned = groups[puser.Group].Is_Banned
		if puser.Is_Banned && puser.Is_Super_Mod {
			puser.Is_Banned = false
		}
		
		if puser.Avatar != "" {
			if puser.Avatar[0] == '.' {
				puser.Avatar = "/uploads/avatar_" + strconv.Itoa(puser.ID) + puser.Avatar
			}
		} else {
			puser.Avatar = strings.Replace(noavatar,"{id}",strconv.Itoa(puser.ID),1)
		}
		
		if groups[puser.Group].Tag != "" {
			puser.Tag = groups[puser.Group].Tag
		} else {
			puser.Tag = ""
		}
		userList = append(userList,puser)
	}
	err = rows.Err()
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	pi := Page{"User Manager",user,noticeList,userList,nil}
	err = templates.ExecuteTemplate(w,"panel-users.html",pi)
	if err != nil {
		InternalError(err,w,r)
	}
}

func route_panel_users_edit(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	
	// Even if they have the right permissions, the control panel is only open to supermods+. There are many areas without subpermissions which assume that the current user is a supermod+ and admins are extremely unlikely to give these permissions to someone who isn't at-least a supermod to begin with
	if !user.Is_Super_Mod || !user.Perms.EditUser {
		NoPermissions(w,r,user)
		return
	}
	
	uid, err := strconv.Atoi(r.URL.Path[len("/panel/users/edit/"):])
	if err != nil {
		LocalError("The provided User ID is not a valid number.",w,r,user)
		return
	}
	
	targetUser, err := users.Get(uid)
	if err == sql.ErrNoRows {
		LocalError("The user you're trying to edit doesn't exist.",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	if targetUser.Is_Admin && !user.Is_Admin {
		LocalError("Only administrators can edit the account of an administrator.",w,r,user)
		return
	}
	
	var groupList []interface{}
	for _, group := range groups[1:] {
		if !user.Perms.EditUserGroupAdmin && group.Is_Admin {
			continue
		}
		if !user.Perms.EditUserGroupSuperMod && group.Is_Mod {
			continue
		}
		groupList = append(groupList,group)
	}
	
	pi := Page{"User Editor",user,noticeList,groupList,targetUser}
	err = templates.ExecuteTemplate(w,"panel-user-edit.html",pi)
	if err != nil {
		InternalError(err,w,r)
	}
}

func route_panel_users_edit_submit(w http.ResponseWriter, r *http.Request){
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditUser {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	tid, err := strconv.Atoi(r.URL.Path[len("/panel/users/edit/submit/"):])
	if err != nil {
		LocalError("The provided User ID is not a valid number.",w,r,user)
		return
	}
	
	targetUser, err := users.Get(tid)
	if err == sql.ErrNoRows {
		LocalError("The user you're trying to edit doesn't exist.",w,r,user)
		return
	} else if err != nil {
		InternalError(err,w,r)
		return
	}
	
	if targetUser.Is_Admin && !user.Is_Admin {
		LocalError("Only administrators can edit the account of an administrator.",w,r,user)
		return
	}
	
	newname := html.EscapeString(r.PostFormValue("user-name"))
	if newname == "" {
		LocalError("You didn't put in a username.", w, r, user)
		return
	}
	
	newemail := html.EscapeString(r.PostFormValue("user-email"))
	if newemail == "" {
		LocalError("You didn't put in an email address.", w, r, user)
		return
	}
	if (newemail != targetUser.Email) && !user.Perms.EditUserEmail {
		LocalError("You need the EditUserEmail permission to edit the email address of a user.",w,r,user)
		return
	}
	
	newpassword := r.PostFormValue("user-password")
	if newpassword != "" && !user.Perms.EditUserPassword {
		LocalError("You need the EditUserPassword permission to edit the password of a user.",w,r,user)
		return
	}
	
	newgroup, err := strconv.Atoi(r.PostFormValue("user-group"))
	if err != nil {
		LocalError("The provided GroupID is not a valid number.",w,r,user)
		return
	}
	
	if (newgroup > groupCapCount) || (newgroup < 0) || groups[newgroup].Name=="" {
		LocalError("The group you're trying to place this user in doesn't exist.",w,r,user)
		return
	}
	
	if !user.Perms.EditUserGroupAdmin && groups[newgroup].Is_Admin {
		LocalError("You need the EditUserGroupAdmin permission to assign someone to an administrator group.",w,r,user)
		return
	}
	if !user.Perms.EditUserGroupSuperMod && groups[newgroup].Is_Mod {
		LocalError("You need the EditUserGroupAdmin permission to assign someone to a super mod group.",w,r,user)
		return
	}
	
	_, err = update_user_stmt.Exec(newname,newemail,newgroup,targetUser.ID)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	if newpassword != "" {
		SetPassword(targetUser.ID, newpassword)
	}
	
	err = users.Load(targetUser.ID)
	if err != nil {
		LocalError("This user no longer exists!",w,r,user)
		return
	}
	
	http.Redirect(w,r,"/panel/users/edit/" + strconv.Itoa(targetUser.ID),http.StatusSeeOther)
}

func route_panel_groups(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod {
		NoPermissions(w,r,user)
		return
	}
	
	var groupList []interface{}
	for _, group := range groups[1:] {
		var rank string
		var rank_emoji string
		var can_edit bool
		var can_delete bool = false
		
		if group.Is_Admin {
			rank = "Admin"
			rank_emoji = "👑"
		} else if group.Is_Mod {
			rank = "Mod"
			rank_emoji = "👮"
		} else if group.Is_Banned {
			rank = "Banned"
			rank_emoji = "⛓️"
		} else if group.ID == 6 {
			rank = "Guest"
			rank_emoji = "👽"
		} else {
			rank = "Member"
			rank_emoji = "👪"
		}
		
		if user.Perms.EditGroup && (!group.Is_Admin || user.Perms.EditGroupAdmin) && (!group.Is_Mod || user.Perms.EditGroupSuperMod) {
			can_edit = true
		} else {
			can_edit = false
		}
		
		groupList = append(groupList, GroupAdmin{group.ID,group.Name,rank,rank_emoji,can_edit,can_delete})
	}
	//fmt.Printf("%+v\n", groupList)
	
	pi := Page{"Group Manager",user,noticeList,groupList,nil}
	templates.ExecuteTemplate(w,"panel-groups.html",pi)
}

func route_panel_groups_edit(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditGroup {
		NoPermissions(w,r,user)
		return
	}
	
	gid, err := strconv.Atoi(r.URL.Path[len("/panel/groups/edit/"):])
	if err != nil {
		LocalError("The Group ID is not a valid integer.",w,r,user)
		return
	}
	
	if !group_exists(gid) {
		//fmt.Println("aaaaa monsters")
		NotFound(w,r)
		return
	}
	
	group := groups[gid]
	if group.Is_Admin && !user.Perms.EditGroupAdmin {
		LocalError("You need the EditGroupAdmin permission to edit an admin group.",w,r,user)
		return
	}
	if group.Is_Mod && !user.Perms.EditGroupSuperMod {
		LocalError("You need the EditGroupSuperMod permission to edit a super-mod group.",w,r,user)
		return
	}
	
	var rank string
	if group.Is_Admin {
		rank = "Admin"
	} else if group.Is_Mod {
		rank = "Mod"
	} else if group.Is_Banned {
		rank = "Banned"
	} else if group.ID == 6 {
		rank = "Guest"
	} else {
		rank = "Member"
	}
	
	var disable_rank bool
	if !user.Perms.EditGroupGlobalPerms || (group.ID == 6) {
		disable_rank = true
	}
	
	pi := EditGroupPage{"Group Editor",user,noticeList,group.ID,group.Name,group.Tag,rank,disable_rank,nil}
	err = templates.ExecuteTemplate(w,"panel-group-edit.html",pi)
	if err != nil {
		InternalError(err,w,r)
	}
}

func route_panel_groups_edit_perms(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditGroup {
		NoPermissions(w,r,user)
		return
	}
	
	gid, err := strconv.Atoi(r.URL.Path[len("/panel/groups/edit/perms/"):])
	if err != nil {
		LocalError("The Group ID is not a valid integer.",w,r,user)
		return
	}
	
	if !group_exists(gid) {
		//fmt.Println("aaaaa monsters")
		NotFound(w,r)
		return
	}
	
	group := groups[gid]
	if group.Is_Admin && !user.Perms.EditGroupAdmin {
		LocalError("You need the EditGroupAdmin permission to edit an admin group.",w,r,user)
		return
	}
	if group.Is_Mod && !user.Perms.EditGroupSuperMod {
		LocalError("You need the EditGroupSuperMod permission to edit a super-mod group.",w,r,user)
		return
	}
	
	var localPerms []NameLangToggle
	localPerms = append(localPerms, NameLangToggle{"ViewTopic",GetLocalPermPhrase("ViewTopic"),group.Perms.ViewTopic})
	localPerms = append(localPerms, NameLangToggle{"LikeItem",GetLocalPermPhrase("LikeItem"),group.Perms.LikeItem})
	localPerms = append(localPerms, NameLangToggle{"CreateTopic",GetLocalPermPhrase("CreateTopic"),group.Perms.CreateTopic})
	//<--
	localPerms = append(localPerms, NameLangToggle{"EditTopic",GetLocalPermPhrase("EditTopic"),group.Perms.EditTopic})
	localPerms = append(localPerms, NameLangToggle{"DeleteTopic",GetLocalPermPhrase("DeleteTopic"),group.Perms.DeleteTopic})
	localPerms = append(localPerms, NameLangToggle{"CreateReply",GetLocalPermPhrase("CreateReply"),group.Perms.CreateReply})
	localPerms = append(localPerms, NameLangToggle{"EditReply",GetLocalPermPhrase("EditReply"),group.Perms.EditReply})
	localPerms = append(localPerms, NameLangToggle{"DeleteReply",GetLocalPermPhrase("DeleteReply"),group.Perms.DeleteReply})
	localPerms = append(localPerms, NameLangToggle{"PinTopic",GetLocalPermPhrase("PinTopic"),group.Perms.PinTopic})
	localPerms = append(localPerms, NameLangToggle{"CloseTopic",GetLocalPermPhrase("CloseTopic"),group.Perms.CloseTopic})
	
	var globalPerms []NameLangToggle
	globalPerms = append(globalPerms, NameLangToggle{"BanUsers",GetGlobalPermPhrase("BanUsers"),group.Perms.BanUsers})
	globalPerms = append(globalPerms, NameLangToggle{"ActivateUsers",GetGlobalPermPhrase("ActivateUsers"),group.Perms.ActivateUsers})
	globalPerms = append(globalPerms, NameLangToggle{"EditUser",GetGlobalPermPhrase("EditUser"),group.Perms.EditUser})
	globalPerms = append(globalPerms, NameLangToggle{"EditUserEmail",GetGlobalPermPhrase("EditUserEmail"),group.Perms.EditUserEmail})
	globalPerms = append(globalPerms, NameLangToggle{"EditUserPassword",GetGlobalPermPhrase("EditUserPassword"),group.Perms.EditUserPassword})
	globalPerms = append(globalPerms, NameLangToggle{"EditUserGroup",GetGlobalPermPhrase("EditUserGroup"),group.Perms.EditUserGroup})
	globalPerms = append(globalPerms, NameLangToggle{"EditUserGroupSuperMod",GetGlobalPermPhrase("EditUserGroupSuperMod"),group.Perms.EditUserGroupSuperMod})
	globalPerms = append(globalPerms, NameLangToggle{"EditUserGroupAdmin",GetGlobalPermPhrase("EditUserGroupAdmin"),group.Perms.EditUserGroupAdmin})
	globalPerms = append(globalPerms, NameLangToggle{"EditGroup",GetGlobalPermPhrase("EditGroup"),group.Perms.EditGroup})
	globalPerms = append(globalPerms, NameLangToggle{"EditGroupLocalPerms",GetGlobalPermPhrase("EditGroupLocalPerms"),group.Perms.EditGroupLocalPerms})
	globalPerms = append(globalPerms, NameLangToggle{"EditGroupGlobalPerms",GetGlobalPermPhrase("EditGroupGlobalPerms"),group.Perms.EditGroupGlobalPerms})
	globalPerms = append(globalPerms, NameLangToggle{"EditGroupSuperMod",GetGlobalPermPhrase("EditGroupSuperMod"),group.Perms.EditGroupSuperMod})
	globalPerms = append(globalPerms, NameLangToggle{"EditGroupAdmin",GetGlobalPermPhrase("EditGroupAdmin"),group.Perms.EditGroupAdmin})
	globalPerms = append(globalPerms, NameLangToggle{"ManageForums",GetGlobalPermPhrase("ManageForums"),group.Perms.ManageForums})
	globalPerms = append(globalPerms, NameLangToggle{"EditSettings",GetGlobalPermPhrase("EditSettings"),group.Perms.EditSettings})
	globalPerms = append(globalPerms, NameLangToggle{"ManageThemes",GetGlobalPermPhrase("ManageThemes"),group.Perms.ManageThemes})
	globalPerms = append(globalPerms, NameLangToggle{"ManagePlugins",GetGlobalPermPhrase("ManagePlugins"),group.Perms.ManagePlugins})
	globalPerms = append(globalPerms, NameLangToggle{"ViewIPs",GetGlobalPermPhrase("ViewIPs"),group.Perms.ViewIPs})
	
	pi := EditGroupPermsPage{"Group Editor",user,noticeList,group.ID,group.Name,localPerms,globalPerms,nil}
	err = templates.ExecuteTemplate(w,"panel-group-edit-perms.html",pi)
	if err != nil {
		InternalError(err,w,r)
	}
}

func route_panel_groups_edit_submit(w http.ResponseWriter, r *http.Request){
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditGroup {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	gid, err := strconv.Atoi(r.URL.Path[len("/panel/groups/edit/submit/"):])
	if err != nil {
		LocalError("The Group ID is not a valid integer.",w,r,user)
		return
	}
	
	if !group_exists(gid) {
		//fmt.Println("aaaaa monsters")
		NotFound(w,r)
		return
	}
	
	group := groups[gid]
	if group.Is_Admin && !user.Perms.EditGroupAdmin {
		LocalError("You need the EditGroupAdmin permission to edit an admin group.",w,r,user)
		return
	}
	if group.Is_Mod && !user.Perms.EditGroupSuperMod {
		LocalError("You need the EditGroupSuperMod permission to edit a super-mod group.",w,r,user)
		return
	}
	
	gname := r.FormValue("group-name")
	if gname == "" {
		LocalError("The group name can't be left blank.",w,r,user)
		return
	}
	gtag := r.FormValue("group-tag")
	rank := r.FormValue("group-type")
	
	var original_rank string
	if group.Is_Admin {
		original_rank = "Admin"
	} else if group.Is_Mod {
		original_rank = "Mod"
	} else if group.Is_Banned {
		original_rank = "Banned"
	} else if group.ID == 6 {
		original_rank = "Guest"
	} else {
		original_rank = "Member"
	}
	
	group_update_mutex.Lock()
	defer group_update_mutex.Unlock()
	if rank != original_rank {
		if !user.Perms.EditGroupGlobalPerms {
			LocalError("You need the EditGroupGlobalPerms permission to change the group type.",w,r,user)
			return
		}
		
		switch(rank) {
			case "Admin":
				if !user.Perms.EditGroupAdmin {
					LocalError("You need the EditGroupAdmin permission to designate this group as an admin group.",w,r,user)
					return
				}
				
				_, err = update_group_rank_stmt.Exec(1,1,0,gid)
				if err != nil {
					InternalError(err,w,r)
					return
				}
				groups[gid].Is_Admin = true
				groups[gid].Is_Mod = true
				groups[gid].Is_Banned = false
			case "Mod":
				if !user.Perms.EditGroupSuperMod {
					LocalError("You need the EditGroupSuperMod permission to designate this group as a super-mod group.",w,r,user)
					return
				}
				
				_, err = update_group_rank_stmt.Exec(0,1,0,gid)
				if err != nil {
					InternalError(err,w,r)
					return
				}
				groups[gid].Is_Admin = false
				groups[gid].Is_Mod = true
				groups[gid].Is_Banned = false
			case "Banned":
				_, err = update_group_rank_stmt.Exec(0,0,1,gid)
				if err != nil {
					InternalError(err,w,r)
					return
				}
				groups[gid].Is_Admin = false
				groups[gid].Is_Mod = false
				groups[gid].Is_Banned = true
			case "Guest":
				LocalError("You can't designate a group as a guest group.",w,r,user)
				return
			case "Member":
				_, err = update_group_rank_stmt.Exec(0,0,0,gid)
				if err != nil {
					InternalError(err,w,r)
					return
				}
				groups[gid].Is_Admin = false
				groups[gid].Is_Mod = false
				groups[gid].Is_Banned = false
			default:
				LocalError("Invalid group type.",w,r,user)
				return
		}
	}
	
	_, err = update_group_stmt.Exec(gname,gtag,gid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	groups[gid].Name = gname
	groups[gid].Tag = gtag
	
	http.Redirect(w,r,"/panel/groups/edit/" + strconv.Itoa(gid),http.StatusSeeOther)
}

func route_panel_groups_edit_perms_submit(w http.ResponseWriter, r *http.Request){
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditGroup {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	gid, err := strconv.Atoi(r.URL.Path[len("/panel/groups/edit/perms/submit/"):])
	if err != nil {
		LocalError("The Group ID is not a valid integer.",w,r,user)
		return
	}
	
	if !group_exists(gid) {
		//fmt.Println("aaaaa monsters")
		NotFound(w,r)
		return
	}
	
	group := groups[gid]
	if group.Is_Admin && !user.Perms.EditGroupAdmin {
		LocalError("You need the EditGroupAdmin permission to edit an admin group.",w,r,user)
		return
	}
	if group.Is_Mod && !user.Perms.EditGroupSuperMod {
		LocalError("You need the EditGroupSuperMod permission to edit a super-mod group.",w,r,user)
		return
	}
	
	//var lpmap map[string]bool = make(map[string]bool)
	var pmap map[string]bool = make(map[string]bool)
	if user.Perms.EditGroupLocalPerms {
		pplist := LocalPermList
		for _, perm := range pplist {
			pvalue := r.PostFormValue("group-perm-" + perm)
			if pvalue == "1" {
				pmap[perm] = true
			} else {
				pmap[perm] = false
			}
		}
	}
	
	//var gpmap map[string]bool = make(map[string]bool)
	if user.Perms.EditGroupGlobalPerms {
		gplist := GlobalPermList
		for _, perm := range gplist {
			pvalue := r.PostFormValue("group-perm-" + perm)
			if pvalue == "1" {
				pmap[perm] = true
			} else {
				pmap[perm] = false
			}
		}
	}
	
	pjson, err := json.Marshal(pmap)
	if err != nil {
		LocalError("Unable to marshal the data",w,r,user)
		return
	}
	
	_, err = update_group_perms_stmt.Exec(pjson,gid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	err = rebuild_group_permissions(gid)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	http.Redirect(w,r,"/panel/groups/edit/perms/" + strconv.Itoa(gid),http.StatusSeeOther)
}

func route_panel_groups_create_submit(w http.ResponseWriter, r *http.Request){
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.EditGroup {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	group_name := r.PostFormValue("group-name")
	if group_name == "" {
		LocalError("You need a name for this group!",w,r,user)
		return
	}
	group_tag := r.PostFormValue("group-tag")
	
	var is_admin bool
	var is_mod bool
	var is_banned bool
	if user.Perms.EditGroupGlobalPerms {
		group_type := r.PostFormValue("group-type")
		if group_type == "Admin" {
			if !user.Perms.EditGroupAdmin {
				LocalError("You need the EditGroupAdmin permission to create admin groups",w,r,user)
				return
			}
			is_admin = true
			is_mod = true
		} else if group_type == "Mod" {
			if !user.Perms.EditGroupSuperMod {
				LocalError("You need the EditGroupSuperMod permission to create admin groups",w,r,user)
				return
			}
			is_mod = true
		} else if group_type == "Banned" {
			is_banned = true
		}
	}
	
	gid, err := create_group(group_name, group_tag, is_admin, is_mod, is_banned)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	fmt.Println(groups)
	http.Redirect(w,r,"/panel/groups/edit/" + strconv.Itoa(gid),http.StatusSeeOther)
}

func route_panel_themes(w http.ResponseWriter, r *http.Request){
	user, noticeList, ok := SessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManageThemes {
		NoPermissions(w,r,user)
		return
	}
	
	var pThemeList []Theme
	var vThemeList []Theme
	for _, theme := range themes {
		if theme.HideFromThemes {
			continue
		}
		if theme.ForkOf == "" {
			pThemeList = append(pThemeList,theme)
		} else {
			vThemeList = append(vThemeList,theme)
		}
		
	}
	
	pi := ThemesPage{"Theme Manager",user,noticeList,pThemeList,vThemeList,nil}
	err := templates.ExecuteTemplate(w,"panel-themes.html",pi)
	if err != nil {
		log.Print(err)
	}
}

func route_panel_themes_default(w http.ResponseWriter, r *http.Request){
	user, ok := SimpleSessionCheck(w,r)
	if !ok {
		return
	}
	if !user.Is_Super_Mod || !user.Perms.ManageThemes {
		NoPermissions(w,r,user)
		return
	}
	if r.FormValue("session") != user.Session {
		SecurityError(w,r,user)
		return
	}
	
	uname := r.URL.Path[len("/panel/themes/default/"):]
	theme, ok := themes[uname]
	if !ok {
		LocalError("The theme isn't registered in the system",w,r,user)
		return
	}
	if theme.Disabled {
		LocalError("You must not enable this theme",w,r,user)
		return
	}
	
	var isDefault bool
	err := db.QueryRow("select `default` from `themes` where `uname` = ?", uname).Scan(&isDefault)
	if err != nil && err != sql.ErrNoRows {
		InternalError(err,w,r)
		return
	}
	
	has_theme := err != sql.ErrNoRows
	if has_theme {
		if isDefault {
			LocalError("The theme is already active",w,r,user)
			return
		}
		_, err = update_theme_stmt.Exec(1,uname)
		if err != nil {
			InternalError(err,w,r)
			return
		}
	} else {
		_, err := add_theme_stmt.Exec(uname,1)
		if err != nil {
			InternalError(err,w,r)
			return
		}
	}
	
	_, err = update_theme_stmt.Exec(0,defaultTheme)
	if err != nil {
		InternalError(err,w,r)
		return
	}
	
	log.Print("Setting theme '" + theme.Name + "' as the default theme")
	theme.Active = true
	themes[uname] = theme
	
	dTheme, ok := themes[defaultTheme]
	if !ok {
		log.Fatal("The default theme is missing")
		return
	}
	dTheme.Active = false
	themes[defaultTheme] = dTheme
	
	defaultTheme = uname
	reset_template_overrides()
	add_theme_static_files(uname)
	map_theme_templates(theme)
	
	http.Redirect(w,r,"/panel/themes/",http.StatusSeeOther)
}
