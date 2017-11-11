package guilds

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"html"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"../../../common"
)

// A blank list to fill out that parameter in Page for routes which don't use it
var tList []interface{}

var ListStmt *sql.Stmt
var MemberListStmt *sql.Stmt
var MemberListJoinStmt *sql.Stmt
var GetMemberStmt *sql.Stmt
var GetGuildStmt *sql.Stmt
var CreateGuildStmt *sql.Stmt
var AttachForumStmt *sql.Stmt
var UnattachForumStmt *sql.Stmt
var AddMemberStmt *sql.Stmt

// Guild is a struct representing a guild
type Guild struct {
	ID      int
	Link    string
	Name    string
	Desc    string
	Active  bool
	Privacy int /* 0: Public, 1: Protected, 2: Private */

	// Who should be able to accept applications and create invites? Mods+ or just admins? Mods is a good start, we can ponder over whether we should make this more flexible in the future.
	Joinable int /* 0: Private, 1: Anyone can join, 2: Applications, 3: Invite-only */

	MemberCount    int
	Owner          int
	Backdrop       string
	CreatedAt      string
	LastUpdateTime string

	MainForumID int
	MainForum   *common.Forum
	Forums      []*common.Forum
	ExtData     common.ExtData
}

type Page struct {
	Title       string
	CurrentUser common.User
	Header      *common.HeaderVars
	ItemList    []*common.TopicsRow
	Forum       *common.Forum
	Guild       *Guild
	Page        int
	LastPage    int
}

// ListPage is a page struct for constructing a list of every guild
type ListPage struct {
	Title       string
	CurrentUser common.User
	Header      *common.HeaderVars
	GuildList   []*Guild
}

type MemberListPage struct {
	Title       string
	CurrentUser common.User
	Header      *common.HeaderVars
	ItemList    []Member
	Guild       *Guild
	Page        int
	LastPage    int
}

// Member is a struct representing a specific member of a guild, not to be confused with the global User struct.
type Member struct {
	Link       string
	Rank       int    /* 0: Member. 1: Mod. 2: Admin. */
	RankString string /* Member, Mod, Admin, Owner */
	PostCount  int
	JoinedAt   string
	Offline    bool // TODO: Need to track the online states of members when WebSockets are enabled

	User common.User
}

func PrebuildTmplList(user common.User, headerVars *common.HeaderVars) common.CTmpl {
	var guildList = []*Guild{
		&Guild{
			ID:             1,
			Name:           "lol",
			Link:           BuildGuildURL(common.NameToSlug("lol"), 1),
			Desc:           "A group for people who like to laugh",
			Active:         true,
			MemberCount:    1,
			Owner:          1,
			CreatedAt:      "date",
			LastUpdateTime: "date",
			MainForumID:    1,
			MainForum:      common.Fstore.DirtyGet(1),
			Forums:         []*common.Forum{common.Fstore.DirtyGet(1)},
		},
	}
	listPage := ListPage{"Guild List", user, headerVars, guildList}
	return common.CTmpl{"guilds_guild_list", "guilds_guild_list.html", "templates/", "guilds.ListPage", listPage}
}

// TODO: Do this properly via the widget system
func CommonAreaWidgets(headerVars *common.HeaderVars) {
	// TODO: Hot Groups? Featured Groups? Official Groups?
	var b bytes.Buffer
	var menu = common.WidgetMenu{"Guilds", []common.WidgetMenuItem{
		common.WidgetMenuItem{"Create Guild", "/guild/create/", false},
	}}

	err := common.Templates.ExecuteTemplate(&b, "widget_menu.html", menu)
	if err != nil {
		common.LogError(err)
		return
	}

	if common.Themes[headerVars.ThemeName].Sidebars == "left" {
		headerVars.Widgets.LeftSidebar = template.HTML(string(b.Bytes()))
	} else if common.Themes[headerVars.ThemeName].Sidebars == "right" || common.Themes[headerVars.ThemeName].Sidebars == "both" {
		headerVars.Widgets.RightSidebar = template.HTML(string(b.Bytes()))
	}
}

// TODO: Do this properly via the widget system
// TODO: Make a better more customisable group widget system
func GuildWidgets(headerVars *common.HeaderVars, guildItem *Guild) (success bool) {
	return false // Disabled until the next commit

	/*var b bytes.Buffer
	var menu WidgetMenu = WidgetMenu{"Guild Options", []WidgetMenuItem{
		WidgetMenuItem{"Join", "/guild/join/" + strconv.Itoa(guildItem.ID), false},
		WidgetMenuItem{"Members", "/guild/members/" + strconv.Itoa(guildItem.ID), false},
	}}

	err := templates.ExecuteTemplate(&b, "widget_menu.html", menu)
	if err != nil {
		common.LogError(err)
		return false
	}

	if themes[headerVars.ThemeName].Sidebars == "left" {
		headerVars.Widgets.LeftSidebar = template.HTML(string(b.Bytes()))
	} else if themes[headerVars.ThemeName].Sidebars == "right" || themes[headerVars.ThemeName].Sidebars == "both" {
		headerVars.Widgets.RightSidebar = template.HTML(string(b.Bytes()))
	} else {
		return false
	}
	return true*/
}

/*
	Custom Pages
*/

func RouteGuildList(w http.ResponseWriter, r *http.Request, user common.User) common.RouteError {
	headerVars, ferr := common.UserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	CommonAreaWidgets(headerVars)

	rows, err := ListStmt.Query()
	if err != nil && err != common.ErrNoRows {
		return common.InternalError(err, w, r)
	}
	defer rows.Close()

	var guildList []*Guild
	for rows.Next() {
		guildItem := &Guild{ID: 0}
		err := rows.Scan(&guildItem.ID, &guildItem.Name, &guildItem.Desc, &guildItem.Active, &guildItem.Privacy, &guildItem.Joinable, &guildItem.Owner, &guildItem.MemberCount, &guildItem.CreatedAt, &guildItem.LastUpdateTime)
		if err != nil {
			return common.InternalError(err, w, r)
		}
		guildItem.Link = BuildGuildURL(common.NameToSlug(guildItem.Name), guildItem.ID)
		guildList = append(guildList, guildItem)
	}
	err = rows.Err()
	if err != nil {
		return common.InternalError(err, w, r)
	}

	pi := ListPage{"Guild List", user, headerVars, guildList}
	err = common.RunThemeTemplate(headerVars.ThemeName, "guilds_guild_list", pi, w)
	if err != nil {
		return common.InternalError(err, w, r)
	}
	return nil
}

func GetGuild(guildID int) (guildItem *Guild, err error) {
	guildItem = &Guild{ID: guildID}
	err = GetGuildStmt.QueryRow(guildID).Scan(&guildItem.Name, &guildItem.Desc, &guildItem.Active, &guildItem.Privacy, &guildItem.Joinable, &guildItem.Owner, &guildItem.MemberCount, &guildItem.MainForumID, &guildItem.Backdrop, &guildItem.CreatedAt, &guildItem.LastUpdateTime)
	return guildItem, err
}

func MiddleViewGuild(w http.ResponseWriter, r *http.Request, user common.User) common.RouteError {
	// SEO URLs...
	halves := strings.Split(r.URL.Path[len("/guild/"):], ".")
	if len(halves) < 2 {
		halves = append(halves, halves[0])
	}
	guildID, err := strconv.Atoi(halves[1])
	if err != nil {
		return common.PreError("Not a valid guild ID", w, r)
	}

	guildItem, err := GetGuild(guildID)
	if err != nil {
		return common.LocalError("Bad guild", w, r, user)
	}
	if !guildItem.Active {
		return common.NotFound(w, r)
	}

	return nil

	// TODO: Re-implement this
	// Re-route the request to routeForums
	//var ctx = context.WithValue(r.Context(), "guilds_current_guild", guildItem)
	//return routeForum(w, r.WithContext(ctx), user, strconv.Itoa(guildItem.MainForumID))
}

func RouteCreateGuild(w http.ResponseWriter, r *http.Request, user common.User) common.RouteError {
	headerVars, ferr := common.UserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}
	// TODO: Add an approval queue mode for group creation
	if !user.Loggedin || !user.PluginPerms["CreateGuild"] {
		return common.NoPermissions(w, r, user)
	}
	CommonAreaWidgets(headerVars)

	pi := common.Page{"Create Guild", user, headerVars, tList, nil}
	err := common.Templates.ExecuteTemplate(w, "guilds_create_guild.html", pi)
	if err != nil {
		return common.InternalError(err, w, r)
	}
	return nil
}

func RouteCreateGuildSubmit(w http.ResponseWriter, r *http.Request, user common.User) common.RouteError {
	// TODO: Add an approval queue mode for group creation
	if !user.Loggedin || !user.PluginPerms["CreateGuild"] {
		return common.NoPermissions(w, r, user)
	}

	var guildActive = true
	var guildName = html.EscapeString(r.PostFormValue("group_name"))
	var guildDesc = html.EscapeString(r.PostFormValue("group_desc"))
	var gprivacy = r.PostFormValue("group_privacy")

	var guildPrivacy int
	switch gprivacy {
	case "0":
		guildPrivacy = 0 // Public
	case "1":
		guildPrivacy = 1 // Protected
	case "2":
		guildPrivacy = 2 // private
	default:
		guildPrivacy = 0
	}

	// Create the backing forum
	fid, err := common.Fstore.Create(guildName, "", true, "")
	if err != nil {
		return common.InternalError(err, w, r)
	}

	res, err := CreateGuildStmt.Exec(guildName, guildDesc, guildActive, guildPrivacy, user.ID, fid)
	if err != nil {
		return common.InternalError(err, w, r)
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return common.InternalError(err, w, r)
	}

	// Add the main backing forum to the forum list
	err = AttachForum(int(lastID), fid)
	if err != nil {
		return common.InternalError(err, w, r)
	}

	_, err = AddMemberStmt.Exec(lastID, user.ID, 2)
	if err != nil {
		return common.InternalError(err, w, r)
	}

	http.Redirect(w, r, BuildGuildURL(common.NameToSlug(guildName), int(lastID)), http.StatusSeeOther)
	return nil
}

func RouteMemberList(w http.ResponseWriter, r *http.Request, user common.User) common.RouteError {
	headerVars, ferr := common.UserCheck(w, r, &user)
	if ferr != nil {
		return ferr
	}

	// SEO URLs...
	halves := strings.Split(r.URL.Path[len("/guild/members/"):], ".")
	if len(halves) < 2 {
		halves = append(halves, halves[0])
	}
	guildID, err := strconv.Atoi(halves[1])
	if err != nil {
		return common.PreError("Not a valid group ID", w, r)
	}

	var guildItem = &Guild{ID: guildID}
	var mainForum int // Unused
	err = GetGuildStmt.QueryRow(guildID).Scan(&guildItem.Name, &guildItem.Desc, &guildItem.Active, &guildItem.Privacy, &guildItem.Joinable, &guildItem.Owner, &guildItem.MemberCount, &mainForum, &guildItem.Backdrop, &guildItem.CreatedAt, &guildItem.LastUpdateTime)
	if err != nil {
		return common.LocalError("Bad group", w, r, user)
	}
	guildItem.Link = BuildGuildURL(common.NameToSlug(guildItem.Name), guildItem.ID)

	GuildWidgets(headerVars, guildItem)

	rows, err := MemberListJoinStmt.Query(guildID)
	if err != nil && err != common.ErrNoRows {
		return common.InternalError(err, w, r)
	}

	var guildMembers []Member
	for rows.Next() {
		guildMember := Member{PostCount: 0}
		err := rows.Scan(&guildMember.User.ID, &guildMember.Rank, &guildMember.PostCount, &guildMember.JoinedAt, &guildMember.User.Name, &guildMember.User.Avatar)
		if err != nil {
			return common.InternalError(err, w, r)
		}
		guildMember.Link = common.BuildProfileURL(common.NameToSlug(guildMember.User.Name), guildMember.User.ID)
		if guildMember.User.Avatar != "" {
			if guildMember.User.Avatar[0] == '.' {
				guildMember.User.Avatar = "/uploads/avatar_" + strconv.Itoa(guildMember.User.ID) + guildMember.User.Avatar
			}
		} else {
			guildMember.User.Avatar = strings.Replace(common.Config.Noavatar, "{id}", strconv.Itoa(guildMember.User.ID), 1)
		}
		guildMember.JoinedAt, _ = common.RelativeTimeFromString(guildMember.JoinedAt)
		if guildItem.Owner == guildMember.User.ID {
			guildMember.RankString = "Owner"
		} else {
			switch guildMember.Rank {
			case 0:
				guildMember.RankString = "Member"
			case 1:
				guildMember.RankString = "Mod"
			case 2:
				guildMember.RankString = "Admin"
			}
		}
		guildMembers = append(guildMembers, guildMember)
	}
	err = rows.Err()
	if err != nil {
		return common.InternalError(err, w, r)
	}
	rows.Close()

	pi := MemberListPage{"Guild Member List", user, headerVars, guildMembers, guildItem, 0, 0}
	// A plugin with plugins. Pluginception!
	if common.PreRenderHooks["pre_render_guilds_member_list"] != nil {
		if common.RunPreRenderHook("pre_render_guilds_member_list", w, r, &user, &pi) {
			return nil
		}
	}
	err = common.RunThemeTemplate(headerVars.ThemeName, "guilds_member_list", pi, w)
	if err != nil {
		return common.InternalError(err, w, r)
	}
	return nil
}

func AttachForum(guildID int, fid int) error {
	_, err := AttachForumStmt.Exec(guildID, fid)
	return err
}

func UnattachForum(fid int) error {
	_, err := AttachForumStmt.Exec(fid)
	return err
}

func BuildGuildURL(slug string, id int) string {
	if slug == "" {
		return "/guild/" + slug + "." + strconv.Itoa(id)
	}
	return "/guild/" + strconv.Itoa(id)
}

/*
	Hooks
*/

// TODO: Prebuild this template
func PreRenderViewForum(w http.ResponseWriter, r *http.Request, user *common.User, data interface{}) (halt bool) {
	pi := data.(*common.ForumPage)
	if pi.Header.ExtData.Items != nil {
		if guildData, ok := pi.Header.ExtData.Items["guilds_current_group"]; ok {
			guildItem := guildData.(*Guild)

			guildpi := Page{pi.Title, pi.CurrentUser, pi.Header, pi.ItemList, pi.Forum, guildItem, pi.Page, pi.LastPage}
			err := common.Templates.ExecuteTemplate(w, "guilds_view_guild.html", guildpi)
			if err != nil {
				common.LogError(err)
				return false
			}
			return true
		}
	}
	return false
}

func TrowAssign(args ...interface{}) interface{} {
	var forum = args[1].(*common.Forum)
	if forum.ParentType == "guild" {
		var topicItem = args[0].(*common.TopicsRow)
		topicItem.ForumLink = "/guild/" + strings.TrimPrefix(topicItem.ForumLink, common.GetForumURLPrefix())
	}
	return nil
}

// TODO: It would be nice, if you could select one of the boards in the group from that drop-down rather than just the one you got linked from
func TopicCreatePreLoop(args ...interface{}) interface{} {
	var fid = args[2].(int)
	if common.Fstore.DirtyGet(fid).ParentType == "guild" {
		var strictmode = args[5].(*bool)
		*strictmode = true
	}
	return nil
}

// TODO: Add privacy options
// TODO: Add support for multiple boards and add per-board simplified permissions
// TODO: Take isJs into account for routes which expect JSON responses
func ForumCheck(args ...interface{}) (skip bool, rerr common.RouteError) {
	var r = args[1].(*http.Request)
	var fid = args[3].(*int)
	var forum = common.Fstore.DirtyGet(*fid)

	if forum.ParentType == "guild" {
		var err error
		var w = args[0].(http.ResponseWriter)
		guildItem, ok := r.Context().Value("guilds_current_group").(*Guild)
		if !ok {
			guildItem, err = GetGuild(forum.ParentID)
			if err != nil {
				return true, common.InternalError(errors.New("Unable to find the parent group for a forum"), w, r)
			}
			if !guildItem.Active {
				return true, common.NotFound(w, r)
			}
			r = r.WithContext(context.WithValue(r.Context(), "guilds_current_group", guildItem))
		}

		var user = args[2].(*common.User)
		var rank int
		var posts int
		var joinedAt string

		// TODO: Group privacy settings. For now, groups are all globally visible

		// Clear the default group permissions
		// TODO: Do this more efficiently, doing it quick and dirty for now to get this out quickly
		common.OverrideForumPerms(&user.Perms, false)
		user.Perms.ViewTopic = true

		err = GetMemberStmt.QueryRow(guildItem.ID, user.ID).Scan(&rank, &posts, &joinedAt)
		if err != nil && err != common.ErrNoRows {
			return true, common.InternalError(err, w, r)
		} else if err != nil {
			// TODO: Should we let admins / guests into public groups?
			return true, common.LocalError("You're not part of this group!", w, r, *user)
		}

		// TODO: Implement bans properly by adding the Local Ban API in the next commit
		// TODO: How does this even work? Refactor it along with the rest of this plugin!
		if rank < 0 {
			return true, common.LocalError("You've been banned from this group!", w, r, *user)
		}

		// Basic permissions for members, more complicated permissions coming in the next commit!
		if guildItem.Owner == user.ID {
			common.OverrideForumPerms(&user.Perms, true)
		} else if rank == 0 {
			user.Perms.LikeItem = true
			user.Perms.CreateTopic = true
			user.Perms.CreateReply = true
		} else {
			common.OverrideForumPerms(&user.Perms, true)
		}
		return true, nil
	}

	return false, nil
}

// TODO: Override redirects? I don't think this is needed quite yet

func Widgets(args ...interface{}) interface{} {
	var zone = args[0].(string)
	var headerVars = args[2].(*common.HeaderVars)
	var request = args[3].(*http.Request)

	if zone != "view_forum" {
		return false
	}

	var forum = args[1].(*common.Forum)
	if forum.ParentType == "guild" {
		// This is why I hate using contexts, all the daisy chains and interface casts x.x
		guildItem, ok := request.Context().Value("guilds_current_group").(*Guild)
		if !ok {
			common.LogError(errors.New("Unable to find a parent group in the context data"))
			return false
		}

		if headerVars.ExtData.Items == nil {
			headerVars.ExtData.Items = make(map[string]interface{})
		}
		headerVars.ExtData.Items["guilds_current_group"] = guildItem

		return GuildWidgets(headerVars, guildItem)
	}
	return false
}
