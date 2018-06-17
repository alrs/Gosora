package main

import "./lib"

func createTables(adapter qgen.Adapter) error {
	qgen.Install.CreateTable("users", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"uid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"name", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"password", "varchar", 100, false, false, ""},

			qgen.DBTableColumn{"salt", "varchar", 80, false, false, "''"},
			qgen.DBTableColumn{"group", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"active", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"is_super_admin", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "createdAt", 0, false, false, ""},
			qgen.DBTableColumn{"lastActiveAt", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"session", "varchar", 200, false, false, "''"},
			//qgen.DBTableColumn{"authToken", "varchar", 200, false, false, "''"},
			qgen.DBTableColumn{"last_ip", "varchar", 200, false, false, "0.0.0.0.0"},
			qgen.DBTableColumn{"email", "varchar", 200, false, false, "''"},
			qgen.DBTableColumn{"avatar", "varchar", 100, false, false, "''"},
			qgen.DBTableColumn{"message", "text", 0, false, false, "''"},
			qgen.DBTableColumn{"url_prefix", "varchar", 20, false, false, "''"},
			qgen.DBTableColumn{"url_name", "varchar", 100, false, false, "''"},
			qgen.DBTableColumn{"level", "smallint", 0, false, false, "0"},
			qgen.DBTableColumn{"score", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"posts", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"bigposts", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"megaposts", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"topics", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"liked", "int", 0, false, false, "0"},

			// These two are to bound liked queries with little bits of information we know about the user to reduce the server load
			qgen.DBTableColumn{"oldestItemLikedCreatedAt", "datetime", 0, false, false, ""}, // For internal use only, semantics may change
			qgen.DBTableColumn{"lastLiked", "datetime", 0, false, false, ""},                // For internal use only, semantics may change

			//qgen.DBTableColumn{"penalty_count","int",0,false,false,"0"},
			qgen.DBTableColumn{"temp_group", "int", 0, false, false, "0"}, // For temporary groups, set this to zero when a temporary group isn't in effect
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"uid", "primary"},
			qgen.DBTableKey{"name", "unique"},
		},
	)

	qgen.Install.CreateTable("users_groups", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"gid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"name", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"permissions", "text", 0, false, false, ""},
			qgen.DBTableColumn{"plugin_perms", "text", 0, false, false, ""},
			qgen.DBTableColumn{"is_mod", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"is_admin", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"is_banned", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"user_count", "int", 0, false, false, "0"}, // TODO: Implement this

			qgen.DBTableColumn{"tag", "varchar", 50, false, false, "''"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"gid", "primary"},
		},
	)

	qgen.Install.CreateTable("users_2fa_keys", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"uid", "int", 0, false, false, ""},
			qgen.DBTableColumn{"secret", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"scratch1", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"scratch2", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"scratch3", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"scratch4", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"scratch5", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"scratch6", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"scratch7", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"scratch8", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"createdAt", "createdAt", 0, false, false, ""},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"uid", "primary"},
		},
	)

	// What should we do about global penalties? Put them on the users table for speed? Or keep them here?
	// Should we add IP Penalties? No, that's a stupid idea, just implement IP Bans properly. What about shadowbans?
	// TODO: Perm overrides
	// TODO: Add a mod-queue and other basic auto-mod features. This is needed for awaiting activation and the mod_queue penalty flag
	// TODO: Add a penalty type where a user is stopped from creating plugin_guilds social groups
	// TODO: Shadow bans. We will probably have a CanShadowBan permission for this, as we *really* don't want people using this lightly.
	/*qgen.Install.CreateTable("users_penalties","","",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"uid","int",0,false,false,""},
			qgen.DBTableColumn{"element_id","int",0,false,false,""},
			qgen.DBTableColumn{"element_type","varchar",50,false,false,""}, //forum, profile?, and social_group. Leave blank for global.
			qgen.DBTableColumn{"overrides","text",0,false,false,"{}"},

			qgen.DBTableColumn{"mod_queue","boolean",0,false,false,"0"},
			qgen.DBTableColumn{"shadow_ban","boolean",0,false,false,"0"},
			qgen.DBTableColumn{"no_avatar","boolean",0,false,false,"0"}, // Coming Soon. Should this be a perm override instead?

			// Do we *really* need rate-limit penalty types? Are we going to be allowing bots or something?
			//qgen.DBTableColumn{"posts_per_hour","int",0,false,false,"0"},
			//qgen.DBTableColumn{"topics_per_hour","int",0,false,false,"0"},
			//qgen.DBTableColumn{"posts_count","int",0,false,false,"0"},
			//qgen.DBTableColumn{"topic_count","int",0,false,false,"0"},
			//qgen.DBTableColumn{"last_hour","int",0,false,false,"0"}, // UNIX Time, as we don't need to do anything too fancy here. When an hour has elapsed since that time, reset the hourly penalty counters.

			qgen.DBTableColumn{"issued_by","int",0,false,false,""},
			qgen.DBTableColumn{"issued_at","createdAt",0,false,false,""},
			qgen.DBTableColumn{"expires_at","datetime",0,false,false,""},
		},
		[]qgen.DBTableKey{},
	)*/

	qgen.Install.CreateTable("users_groups_scheduler", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"uid", "int", 0, false, false, ""},
			qgen.DBTableColumn{"set_group", "int", 0, false, false, ""},

			qgen.DBTableColumn{"issued_by", "int", 0, false, false, ""},
			qgen.DBTableColumn{"issued_at", "createdAt", 0, false, false, ""},
			qgen.DBTableColumn{"revert_at", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"temporary", "boolean", 0, false, false, ""}, // special case for permanent bans to do the necessary bookkeeping, might be removed in the future
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"uid", "primary"},
		},
	)

	qgen.Install.CreateTable("emails", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"email", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"uid", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"validated", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"token", "varchar", 200, false, false, "''"},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("forums", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"fid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"name", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"desc", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"active", "boolean", 0, false, false, "1"},
			qgen.DBTableColumn{"topicCount", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"preset", "varchar", 100, false, false, "''"},
			qgen.DBTableColumn{"parentID", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"parentType", "varchar", 50, false, false, "''"},
			qgen.DBTableColumn{"lastTopicID", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"lastReplyerID", "int", 0, false, false, "0"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"fid", "primary"},
		},
	)

	qgen.Install.CreateTable("forums_permissions", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"fid", "int", 0, false, false, ""},
			qgen.DBTableColumn{"gid", "int", 0, false, false, ""},
			qgen.DBTableColumn{"preset", "varchar", 100, false, false, "''"},
			qgen.DBTableColumn{"permissions", "text", 0, false, false, ""},
		},
		[]qgen.DBTableKey{
			// TODO: Test to see that the compound primary key works
			qgen.DBTableKey{"fid,gid", "primary"},
		},
	)

	qgen.Install.CreateTable("topics", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"tid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"title", "varchar", 100, false, false, ""}, // TODO: Increase the max length to 200?
			qgen.DBTableColumn{"content", "text", 0, false, false, ""},
			qgen.DBTableColumn{"parsed_content", "text", 0, false, false, ""},
			qgen.DBTableColumn{"createdAt", "createdAt", 0, false, false, ""},
			qgen.DBTableColumn{"lastReplyAt", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"lastReplyBy", "int", 0, false, false, ""},
			qgen.DBTableColumn{"createdBy", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"is_closed", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"sticky", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"parentID", "int", 0, false, false, "2"},
			qgen.DBTableColumn{"ipaddress", "varchar", 200, false, false, "0.0.0.0.0"},
			qgen.DBTableColumn{"postCount", "int", 0, false, false, "1"},
			qgen.DBTableColumn{"likeCount", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"words", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"views", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"css_class", "varchar", 100, false, false, "''"},
			qgen.DBTableColumn{"poll", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"data", "varchar", 200, false, false, "''"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"tid", "primary"},
		},
	)

	qgen.Install.CreateTable("replies", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"rid", "int", 0, false, true, ""},  // TODO: Rename to replyID?
			qgen.DBTableColumn{"tid", "int", 0, false, false, ""}, // TODO: Rename to topicID?
			qgen.DBTableColumn{"content", "text", 0, false, false, ""},
			qgen.DBTableColumn{"parsed_content", "text", 0, false, false, ""},
			qgen.DBTableColumn{"createdAt", "createdAt", 0, false, false, ""},
			qgen.DBTableColumn{"createdBy", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"lastEdit", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"lastEditBy", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"lastUpdated", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"ipaddress", "varchar", 200, false, false, "0.0.0.0.0"},
			qgen.DBTableColumn{"likeCount", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"words", "int", 0, false, false, "1"}, // ? - replies has a default of 1 and topics has 0? why?
			qgen.DBTableColumn{"actionType", "varchar", 20, false, false, "''"},
			qgen.DBTableColumn{"poll", "int", 0, false, false, "0"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"rid", "primary"},
		},
	)

	qgen.Install.CreateTable("attachments", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"attachID", "int", 0, false, true, ""},
			qgen.DBTableColumn{"sectionID", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"sectionTable", "varchar", 200, false, false, "forums"},
			qgen.DBTableColumn{"originID", "int", 0, false, false, ""},
			qgen.DBTableColumn{"originTable", "varchar", 200, false, false, "replies"},
			qgen.DBTableColumn{"uploadedBy", "int", 0, false, false, ""}, // TODO; Make this a foreign key
			qgen.DBTableColumn{"path", "varchar", 200, false, false, ""},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"attachID", "primary"},
		},
	)

	qgen.Install.CreateTable("revisions", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"reviseID", "int", 0, false, true, ""},
			qgen.DBTableColumn{"content", "text", 0, false, false, ""},
			qgen.DBTableColumn{"contentID", "int", 0, false, false, ""},
			qgen.DBTableColumn{"contentType", "varchar", 100, false, false, "replies"},
			qgen.DBTableColumn{"createdAt", "createdAt", 0, false, false, ""},
			// TODO: Add a createdBy column?
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"reviseID", "primary"},
		},
	)

	qgen.Install.CreateTable("polls", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"pollID", "int", 0, false, true, ""},
			qgen.DBTableColumn{"parentID", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"parentTable", "varchar", 100, false, false, "topics"}, // topics, replies
			qgen.DBTableColumn{"type", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"options", "json", 0, false, false, ""},
			qgen.DBTableColumn{"votes", "int", 0, false, false, "0"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"pollID", "primary"},
		},
	)

	qgen.Install.CreateTable("polls_options", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"pollID", "int", 0, false, false, ""},
			qgen.DBTableColumn{"option", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"votes", "int", 0, false, false, "0"},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("polls_votes", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"pollID", "int", 0, false, false, ""},
			qgen.DBTableColumn{"uid", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"option", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"castAt", "createdAt", 0, false, false, ""},
			qgen.DBTableColumn{"ipaddress", "varchar", 200, false, false, "0.0.0.0.0"},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("users_replies", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"rid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"uid", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"content", "text", 0, false, false, ""},
			qgen.DBTableColumn{"parsed_content", "text", 0, false, false, ""},
			qgen.DBTableColumn{"createdAt", "createdAt", 0, false, false, ""},
			qgen.DBTableColumn{"createdBy", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"lastEdit", "int", 0, false, false, ""},
			qgen.DBTableColumn{"lastEditBy", "int", 0, false, false, ""},
			qgen.DBTableColumn{"ipaddress", "varchar", 200, false, false, "0.0.0.0.0"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"rid", "primary"},
		},
	)

	qgen.Install.CreateTable("likes", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"weight", "tinyint", 0, false, false, "1"},
			qgen.DBTableColumn{"targetItem", "int", 0, false, false, ""},
			qgen.DBTableColumn{"targetType", "varchar", 50, false, false, "replies"},
			qgen.DBTableColumn{"sentBy", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"createdAt", "createdAt", 0, false, false, ""},
			qgen.DBTableColumn{"recalc", "tinyint", 0, false, false, "0"},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("activity_stream_matches", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"watcher", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"asid", "int", 0, false, false, ""},    // TODO: Make this a foreign key
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("activity_stream", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"asid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"actor", "int", 0, false, false, ""},            /* the one doing the act */ // TODO: Make this a foreign key
			qgen.DBTableColumn{"targetUser", "int", 0, false, false, ""},       /* the user who created the item the actor is acting on, some items like forums may lack a targetUser field */
			qgen.DBTableColumn{"event", "varchar", 50, false, false, ""},       /* mention, like, reply (as in the act of replying to an item, not the reply item type, you can "reply" to a forum by making a topic in it), friend_invite */
			qgen.DBTableColumn{"elementType", "varchar", 50, false, false, ""}, /* topic, post (calling it post here to differentiate it from the 'reply' event), forum, user */
			qgen.DBTableColumn{"elementID", "int", 0, false, false, ""},        /* the ID of the element being acted upon */
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"asid", "primary"},
		},
	)

	qgen.Install.CreateTable("activity_subscriptions", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"user", "int", 0, false, false, ""},            // TODO: Make this a foreign key
			qgen.DBTableColumn{"targetID", "int", 0, false, false, ""},        /* the ID of the element being acted upon */
			qgen.DBTableColumn{"targetType", "varchar", 50, false, false, ""}, /* topic, post (calling it post here to differentiate it from the 'reply' event), forum, user */
			qgen.DBTableColumn{"level", "int", 0, false, false, "0"},          /* 0: Mentions (aka the global default for any post), 1: Replies To You, 2: All Replies*/
		},
		[]qgen.DBTableKey{},
	)

	/* Due to MySQL's design, we have to drop the unique keys for table settings, plugins, and themes down from 200 to 180 or it will error */
	qgen.Install.CreateTable("settings", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"name", "varchar", 180, false, false, ""},
			qgen.DBTableColumn{"content", "varchar", 250, false, false, ""},
			qgen.DBTableColumn{"type", "varchar", 50, false, false, ""},
			qgen.DBTableColumn{"constraints", "varchar", 200, false, false, "''"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"name", "unique"},
		},
	)

	qgen.Install.CreateTable("word_filters", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"wfid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"find", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"replacement", "varchar", 200, false, false, ""},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"wfid", "primary"},
		},
	)

	qgen.Install.CreateTable("plugins", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"uname", "varchar", 180, false, false, ""},
			qgen.DBTableColumn{"active", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"installed", "boolean", 0, false, false, "0"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"uname", "unique"},
		},
	)

	qgen.Install.CreateTable("themes", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"uname", "varchar", 180, false, false, ""},
			qgen.DBTableColumn{"default", "boolean", 0, false, false, "0"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"uname", "unique"},
		},
	)

	qgen.Install.CreateTable("widgets", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"position", "int", 0, false, false, ""},
			qgen.DBTableColumn{"side", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"type", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"active", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"location", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"data", "text", 0, false, false, "''"},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("menus", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"mid", "int", 0, false, true, ""},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"mid", "primary"},
		},
	)

	qgen.Install.CreateTable("menu_items", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"miid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"mid", "int", 0, false, false, ""},
			qgen.DBTableColumn{"name", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"htmlID", "varchar", 200, false, false, "''"},
			qgen.DBTableColumn{"cssClass", "varchar", 200, false, false, "''"},
			qgen.DBTableColumn{"position", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"path", "varchar", 200, false, false, "''"},
			qgen.DBTableColumn{"aria", "varchar", 200, false, false, "''"},
			qgen.DBTableColumn{"tooltip", "varchar", 200, false, false, "''"},
			qgen.DBTableColumn{"tmplName", "varchar", 200, false, false, "''"},
			qgen.DBTableColumn{"order", "int", 0, false, false, "0"},

			qgen.DBTableColumn{"guestOnly", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"memberOnly", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"staffOnly", "boolean", 0, false, false, "0"},
			qgen.DBTableColumn{"adminOnly", "boolean", 0, false, false, "0"},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"miid", "primary"},
		},
	)

	qgen.Install.CreateTable("pages", "utf8mb4", "utf8mb4_general_ci",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"pid", "int", 0, false, true, ""},
			//qgen.DBTableColumn{"path", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"name", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"title", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"body", "text", 0, false, false, ""},
			// TODO: Make this a table?
			qgen.DBTableColumn{"allowedGroups", "text", 0, false, false, ""},
			qgen.DBTableColumn{"menuID", "int", 0, false, false, "-1"}, // simple sidebar menu
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"pid", "primary"},
		},
	)

	qgen.Install.CreateTable("registration_logs", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"rlid", "int", 0, false, true, ""},
			qgen.DBTableColumn{"username", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"email", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"failureReason", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"success", "bool", 0, false, false, "0"}, // Did this attempt succeed?
			qgen.DBTableColumn{"ipaddress", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"doneAt", "createdAt", 0, false, false, ""},
		},
		[]qgen.DBTableKey{
			qgen.DBTableKey{"rlid", "primary"},
		},
	)

	qgen.Install.CreateTable("moderation_logs", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"action", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"elementID", "int", 0, false, false, ""},
			qgen.DBTableColumn{"elementType", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"ipaddress", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"actorID", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"doneAt", "datetime", 0, false, false, ""},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("administration_logs", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"action", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"elementID", "int", 0, false, false, ""},
			qgen.DBTableColumn{"elementType", "varchar", 100, false, false, ""},
			qgen.DBTableColumn{"ipaddress", "varchar", 200, false, false, ""},
			qgen.DBTableColumn{"actorID", "int", 0, false, false, ""}, // TODO: Make this a foreign key
			qgen.DBTableColumn{"doneAt", "datetime", 0, false, false, ""},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("viewchunks", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"count", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"route", "varchar", 200, false, false, ""},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("viewchunks_agents", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"count", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"browser", "varchar", 200, false, false, ""}, // googlebot, firefox, opera, etc.
			//qgen.DBTableColumn{"version","varchar",0,false,false,""}, // the version of the browser or bot
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("viewchunks_systems", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"count", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"system", "varchar", 200, false, false, ""}, // windows, android, unknown, etc.
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("viewchunks_langs", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"count", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"lang", "varchar", 200, false, false, ""}, // en, ru, etc.
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("viewchunks_referrers", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"count", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"domain", "varchar", 200, false, false, ""},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("viewchunks_forums", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"count", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "datetime", 0, false, false, ""},
			qgen.DBTableColumn{"forum", "int", 0, false, false, ""},
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("topicchunks", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"count", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "datetime", 0, false, false, ""},
			// TODO: Add a column for the parent forum?
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("postchunks", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"count", "int", 0, false, false, "0"},
			qgen.DBTableColumn{"createdAt", "datetime", 0, false, false, ""},
			// TODO: Add a column for the parent topic / profile?
		},
		[]qgen.DBTableKey{},
	)

	qgen.Install.CreateTable("sync", "", "",
		[]qgen.DBTableColumn{
			qgen.DBTableColumn{"last_update", "datetime", 0, false, false, ""},
		},
		[]qgen.DBTableKey{},
	)

	return nil
}
