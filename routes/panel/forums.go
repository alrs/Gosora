package panel

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	c "github.com/Azareal/Gosora/common"
	p "github.com/Azareal/Gosora/common/phrases"
)

func Forums(w http.ResponseWriter, r *http.Request, user c.User) c.RouteError {
	basePage, ferr := buildBasePage(w, r, &user, "forums", "forums")
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}
	basePage.Header.AddScript("Sortable-1.4.0/Sortable.min.js")
	basePage.Header.AddScriptAsync("panel_forums.js")

	// TODO: Paginate this?
	var forumList []interface{}
	forums, err := c.Forums.GetAll()
	if err != nil {
		return c.InternalError(err, w, r)
	}

	// ? - Should we generate something similar to the forumView? It might be a little overkill for a page which is rarely loaded in comparison to /forums/
	for _, f := range forums {
		if f.Name != "" && f.ParentID == 0 {
			fadmin := c.ForumAdmin{f.ID, f.Name, f.Desc, f.Active, f.Preset, f.TopicCount, c.PresetToLang(f.Preset)}
			if fadmin.Preset == "" {
				fadmin.Preset = "custom"
			}
			forumList = append(forumList, fadmin)
		}
	}

	if r.FormValue("created") == "1" {
		basePage.AddNotice("panel_forum_created")
	} else if r.FormValue("deleted") == "1" {
		basePage.AddNotice("panel_forum_deleted")
	} else if r.FormValue("updated") == "1" {
		basePage.AddNotice("panel_forum_updated")
	}

	pi := c.PanelPage{basePage, forumList, nil}
	return renderTemplate("panel", w, r, basePage.Header, c.Panel{basePage, "", "", "panel_forums", &pi})
}

func ForumsCreateSubmit(w http.ResponseWriter, r *http.Request, user c.User) c.RouteError {
	_, ferr := c.SimplePanelUserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}

	fname := r.PostFormValue("name")
	fdesc := r.PostFormValue("desc")
	fpreset := c.StripInvalidPreset(r.PostFormValue("preset"))
	factive := r.PostFormValue("active")
	active := (factive == "on" || factive == "1")

	fid, err := c.Forums.Create(fname, fdesc, active, fpreset)
	if err != nil {
		return c.InternalError(err, w, r)
	}
	err = c.AdminLogs.Create("create", fid, "forum", user.GetIP(), user.ID)
	if err != nil {
		return c.InternalError(err, w, r)
	}

	http.Redirect(w, r, "/panel/forums/?created=1", http.StatusSeeOther)
	return nil
}

// TODO: Revamp this
func ForumsDelete(w http.ResponseWriter, r *http.Request, user c.User, sfid string) c.RouteError {
	basePage, ferr := buildBasePage(w, r, &user, "delete_forum", "forums")
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}

	fid, err := strconv.Atoi(sfid)
	if err != nil {
		return c.LocalError("The provided Forum ID is not a valid number.", w, r, user)
	}
	forum, err := c.Forums.Get(fid)
	if err == sql.ErrNoRows {
		return c.LocalError("The forum you're trying to delete doesn't exist.", w, r, user)
	} else if err != nil {
		return c.InternalError(err, w, r)
	}

	confirmMsg := p.GetTmplPhrasef("panel_forum_delete_are_you_sure", forum.Name)
	yousure := c.AreYouSure{"/panel/forums/delete/submit/" + strconv.Itoa(fid), confirmMsg}

	pi := c.PanelPage{basePage, tList, yousure}
	if c.RunPreRenderHook("pre_render_panel_delete_forum", w, r, &user, &pi) {
		return nil
	}
	return renderTemplate("panel_are_you_sure", w, r, basePage.Header, &pi)
}

func ForumsDeleteSubmit(w http.ResponseWriter, r *http.Request, user c.User, sfid string) c.RouteError {
	_, ferr := c.SimplePanelUserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}

	fid, err := strconv.Atoi(sfid)
	if err != nil {
		return c.LocalError("The provided Forum ID is not a valid number.", w, r, user)
	}
	err = c.Forums.Delete(fid)
	if err == sql.ErrNoRows {
		return c.LocalError("The forum you're trying to delete doesn't exist.", w, r, user)
	} else if err != nil {
		return c.InternalError(err, w, r)
	}
	err = c.AdminLogs.Create("delete", fid, "forum", user.GetIP(), user.ID)
	if err != nil {
		return c.InternalError(err, w, r)
	}

	http.Redirect(w, r, "/panel/forums/?deleted=1", http.StatusSeeOther)
	return nil
}

func ForumsOrderSubmit(w http.ResponseWriter, r *http.Request, user c.User) c.RouteError {
	_, ferr := c.SimplePanelUserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	js := r.PostFormValue("js") == "1"
	if !user.Perms.ManageForums {
		return c.NoPermissionsJSQ(w, r, user, js)
	}
	sitems := strings.TrimSuffix(strings.TrimPrefix(r.PostFormValue("items"), "{"), "}")
	//fmt.Printf("sitems: %+v\n", sitems)

	updateMap := make(map[int]int)
	for index, sfid := range strings.Split(sitems, ",") {
		fid, err := strconv.Atoi(sfid)
		if err != nil {
			return c.LocalErrorJSQ("Invalid integer in forum list", w, r, user, js)
		}
		updateMap[fid] = index
	}
	c.Forums.UpdateOrder(updateMap)

	err := c.AdminLogs.Create("reorder", 0, "forum", user.GetIP(), user.ID)
	if err != nil {
		return c.InternalErrorJSQ(err, w, r, js)
	}

	return successRedirect("/panel/forums/", w, r, js)
}

func ForumsEdit(w http.ResponseWriter, r *http.Request, user c.User, sfid string) c.RouteError {
	basePage, ferr := buildBasePage(w, r, &user, "edit_forum", "forums")
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}

	fid, err := strconv.Atoi(sfid)
	if err != nil {
		return c.SimpleError(p.GetErrorPhrase("url_id_must_be_integer"), w, r, basePage.Header)
	}
	basePage.Header.AddScriptAsync("panel_forum_edit.js")

	forum, err := c.Forums.Get(fid)
	if err == sql.ErrNoRows {
		return c.LocalError("The forum you're trying to edit doesn't exist.", w, r, user)
	} else if err != nil {
		return c.InternalError(err, w, r)
	}
	if forum.Preset == "" {
		forum.Preset = "custom"
	}

	glist, err := c.Groups.GetAll()
	if err != nil {
		return c.InternalError(err, w, r)
	}

	var gplist []c.GroupForumPermPreset
	for gid, group := range glist {
		if gid == 0 {
			continue
		}
		forumPerms, err := c.FPStore.Get(fid, group.ID)
		if err == sql.ErrNoRows {
			forumPerms = c.BlankForumPerms()
		} else if err != nil {
			return c.InternalError(err, w, r)
		}
		preset := c.ForumPermsToGroupForumPreset(forumPerms)
		gplist = append(gplist, c.GroupForumPermPreset{group, preset, preset == "default"})
	}

	if r.FormValue("updated") == "1" {
		basePage.AddNotice("panel_forum_updated")
	}

	pi := c.PanelEditForumPage{basePage, forum.ID, forum.Name, forum.Desc, forum.Active, forum.Preset, gplist}
	return renderTemplate("panel", w, r, basePage.Header, c.Panel{basePage, "", "", "panel_forum_edit", &pi})
}

func ForumsEditSubmit(w http.ResponseWriter, r *http.Request, user c.User, sfid string) c.RouteError {
	_, ferr := c.SimplePanelUserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}
	js := r.PostFormValue("js") == "1"

	fid, err := strconv.Atoi(sfid)
	if err != nil {
		return c.LocalErrorJSQ("The provided Forum ID is not a valid number.", w, r, user, js)
	}
	forum, err := c.Forums.Get(fid)
	if err == sql.ErrNoRows {
		return c.LocalErrorJSQ("The forum you're trying to edit doesn't exist.", w, r, user, js)
	} else if err != nil {
		return c.InternalErrorJSQ(err, w, r, js)
	}

	fname := r.PostFormValue("forum_name")
	fdesc := r.PostFormValue("forum_desc")
	fpreset := c.StripInvalidPreset(r.PostFormValue("forum_preset"))
	factive := r.PostFormValue("forum_active")

	active := false
	if factive == "" {
		active = forum.Active
	} else if factive == "1" || factive == "Show" {
		active = true
	}

	err = forum.Update(fname, fdesc, active, fpreset)
	if err != nil {
		return c.InternalErrorJSQ(err, w, r, js)
	}
	err = c.AdminLogs.Create("edit", fid, "forum", user.GetIP(), user.ID)
	if err != nil {
		return c.InternalError(err, w, r)
	}

	// ? Should we redirect to the forum editor instead?
	return successRedirect("/panel/forums/", w, r, js)
}

func ForumsEditPermsSubmit(w http.ResponseWriter, r *http.Request, user c.User, sfid string) c.RouteError {
	_, ferr := c.SimplePanelUserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}
	js := r.PostFormValue("js") == "1"

	fid, err := strconv.Atoi(sfid)
	if err != nil {
		return c.LocalErrorJSQ("The provided Forum ID is not a valid number.", w, r, user, js)
	}

	gid, err := strconv.Atoi(r.PostFormValue("gid"))
	if err != nil {
		return c.LocalErrorJSQ("Invalid Group ID", w, r, user, js)
	}

	forum, err := c.Forums.Get(fid)
	if err == sql.ErrNoRows {
		return c.LocalErrorJSQ("This forum doesn't exist", w, r, user, js)
	} else if err != nil {
		return c.InternalErrorJSQ(err, w, r, js)
	}

	permPreset := c.StripInvalidGroupForumPreset(r.PostFormValue("perm_preset"))
	err = forum.SetPreset(permPreset, gid)
	if err != nil {
		return c.LocalErrorJSQ(err.Error(), w, r, user, js)
	}
	err = c.AdminLogs.Create("edit", fid, "forum", user.GetIP(), user.ID)
	if err != nil {
		return c.InternalError(err, w, r)
	}

	return successRedirect("/panel/forums/edit/"+strconv.Itoa(fid)+"?updated=1", w, r, js)
}

// A helper function for the Advanced portion of the Forum Perms Editor
func forumPermsExtractDash(paramList string) (fid int, gid int, err error) {
	params := strings.Split(paramList, "-")
	if len(params) != 2 {
		return fid, gid, errors.New("Parameter count mismatch")
	}

	fid, err = strconv.Atoi(params[0])
	if err != nil {
		return fid, gid, errors.New("The provided Forum ID is not a valid number.")
	}

	gid, err = strconv.Atoi(params[1])
	if err != nil {
		err = errors.New("The provided Group ID is not a valid number.")
	}

	return fid, gid, err
}

func ForumsEditPermsAdvance(w http.ResponseWriter, r *http.Request, user c.User, paramList string) c.RouteError {
	basePage, ferr := buildBasePage(w, r, &user, "edit_forum", "forums")
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}

	fid, gid, err := forumPermsExtractDash(paramList)
	if err != nil {
		return c.LocalError(err.Error(), w, r, user)
	}

	forum, err := c.Forums.Get(fid)
	if err == sql.ErrNoRows {
		return c.LocalError("The forum you're trying to edit doesn't exist.", w, r, user)
	} else if err != nil {
		return c.InternalError(err, w, r)
	}
	if forum.Preset == "" {
		forum.Preset = "custom"
	}

	fp, err := c.FPStore.Get(fid, gid)
	if err == sql.ErrNoRows {
		fp = c.BlankForumPerms()
	} else if err != nil {
		return c.InternalError(err, w, r)
	}

	var formattedPermList []c.NameLangToggle
	// TODO: Load the phrases in bulk for efficiency?
	// TODO: Reduce the amount of code duplication between this and the group editor. Also, can we grind this down into one line or use a code generator to stay current more easily?
	addToggle := func(permStr string, perm bool) {
		formattedPermList = append(formattedPermList, c.NameLangToggle{permStr, p.GetPermPhrase(permStr), perm})
	}
	addToggle("ViewTopic", fp.ViewTopic)
	addToggle("LikeItem", fp.LikeItem)
	addToggle("CreateTopic", fp.CreateTopic)
	//<--
	addToggle("EditTopic", fp.EditTopic)
	addToggle("DeleteTopic", fp.DeleteTopic)
	addToggle("CreateReply", fp.CreateReply)
	addToggle("EditReply", fp.EditReply)
	addToggle("DeleteReply", fp.DeleteReply)
	addToggle("PinTopic", fp.PinTopic)
	addToggle("CloseTopic", fp.CloseTopic)
	addToggle("MoveTopic", fp.MoveTopic)

	if r.FormValue("updated") == "1" {
		basePage.AddNotice("panel_forum_perms_updated")
	}

	pi := c.PanelEditForumGroupPage{basePage, forum.ID, gid, forum.Name, forum.Desc, forum.Active, forum.Preset, formattedPermList}
	return renderTemplate("panel", w, r, basePage.Header, c.Panel{basePage, "", "", "panel_forum_edit_perms", &pi})
}

func ForumsEditPermsAdvanceSubmit(w http.ResponseWriter, r *http.Request, user c.User, paramList string) c.RouteError {
	_, ferr := c.SimplePanelUserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	if !user.Perms.ManageForums {
		return c.NoPermissions(w, r, user)
	}
	js := r.PostFormValue("js") == "1"

	fid, gid, err := forumPermsExtractDash(paramList)
	if err != nil {
		return c.LocalError(err.Error(), w, r, user)
	}

	forum, err := c.Forums.Get(fid)
	if err == sql.ErrNoRows {
		return c.LocalError("The forum you're trying to edit doesn't exist.", w, r, user)
	} else if err != nil {
		return c.InternalError(err, w, r)
	}

	fp, err := c.FPStore.GetCopy(fid, gid)
	if err == sql.ErrNoRows {
		fp = *c.BlankForumPerms()
	} else if err != nil {
		return c.InternalError(err, w, r)
	}

	extractPerm := func(name string) bool {
		pvalue := r.PostFormValue("forum-perm-" + name)
		return (pvalue == "1")
	}

	// TODO: Generate this code?
	fp.ViewTopic = extractPerm("ViewTopic")
	fp.LikeItem = extractPerm("LikeItem")
	fp.CreateTopic = extractPerm("CreateTopic")
	fp.EditTopic = extractPerm("EditTopic")
	fp.DeleteTopic = extractPerm("DeleteTopic")
	fp.CreateReply = extractPerm("CreateReply")
	fp.EditReply = extractPerm("EditReply")
	fp.DeleteReply = extractPerm("DeleteReply")
	fp.PinTopic = extractPerm("PinTopic")
	fp.CloseTopic = extractPerm("CloseTopic")
	fp.MoveTopic = extractPerm("MoveTopic")

	err = forum.SetPerms(&fp, "custom", gid)
	if err != nil {
		return c.LocalErrorJSQ(err.Error(), w, r, user, js)
	}
	err = c.AdminLogs.Create("edit", fid, "forum", user.GetIP(), user.ID)
	if err != nil {
		return c.InternalError(err, w, r)
	}

	return successRedirect("/panel/forums/edit/perms/"+strconv.Itoa(fid)+"-"+strconv.Itoa(gid)+"?updated=1", w, r, js)
}
