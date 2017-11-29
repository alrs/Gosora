INSERT INTO [sync] ([last_update]) VALUES (GETUTCDATE());
INSERT INTO [settings] ([name],[content],[type]) VALUES ('url_tags','1','bool');
INSERT INTO [settings] ([name],[content],[type],[constraints]) VALUES ('activation_type','1','list','1-3');
INSERT INTO [settings] ([name],[content],[type]) VALUES ('bigpost_min_words','250','int');
INSERT INTO [settings] ([name],[content],[type]) VALUES ('megapost_min_words','1000','int');
INSERT INTO [themes] ([uname],[default]) VALUES ('tempra-simple',1);
INSERT INTO [emails] ([email],[uid],[validated]) VALUES ('admin@localhost',1,1);
INSERT INTO [users_groups] ([name],[permissions],[plugin_perms],[is_mod],[is_admin],[tag]) VALUES ('Administrator','{"BanUsers":true,"ActivateUsers":true,"EditUser":true,"EditUserEmail":true,"EditUserPassword":true,"EditUserGroup":true,"EditUserGroupSuperMod":true,"EditUserGroupAdmin":false,"EditGroup":true,"EditGroupLocalPerms":true,"EditGroupGlobalPerms":true,"EditGroupSuperMod":true,"EditGroupAdmin":false,"ManageForums":true,"EditSettings":true,"ManageThemes":true,"ManagePlugins":true,"ViewAdminLogs":true,"ViewIPs":true,"UploadFiles":true,"ViewTopic":true,"LikeItem":true,"CreateTopic":true,"EditTopic":true,"DeleteTopic":true,"CreateReply":true,"EditReply":true,"DeleteReply":true,"PinTopic":true,"CloseTopic":true}','{}',1,1,'Admin');
INSERT INTO [users_groups] ([name],[permissions],[plugin_perms],[is_mod],[tag]) VALUES ('Moderator','{"BanUsers":true,"ActivateUsers":false,"EditUser":true,"EditUserEmail":false,"EditUserGroup":true,"ViewIPs":true,"UploadFiles":true,"ViewTopic":true,"LikeItem":true,"CreateTopic":true,"EditTopic":true,"DeleteTopic":true,"CreateReply":true,"EditReply":true,"DeleteReply":true,"PinTopic":true,"CloseTopic":true}','{}',1,'Mod');
INSERT INTO [users_groups] ([name],[permissions],[plugin_perms]) VALUES ('Member','{"UploadFiles":true,"ViewTopic":true,"LikeItem":true,"CreateTopic":true,"CreateReply":true}','{}');
INSERT INTO [users_groups] ([name],[permissions],[plugin_perms],[is_banned]) VALUES ('Banned','{"ViewTopic":true}','{}',1);
INSERT INTO [users_groups] ([name],[permissions],[plugin_perms]) VALUES ('AwaitingActivation','{"ViewTopic":true}','{}');
INSERT INTO [users_groups] ([name],[permissions],[plugin_perms],[tag]) VALUES ('NotLoggedin','{"ViewTopic":true}','{}','Guest');
INSERT INTO [forums] ([name],[active],[desc]) VALUES ('Reports',0,'Allthereportsgohere');
INSERT INTO [forums] ([name],[lastTopicID],[lastReplyerID],[desc]) VALUES ('General',1,1,'Aplaceforgeneraldiscussionswhichdon''tfitelsewhere');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (1,1,'{"ViewTopic":true,"CreateReply":true,"CreateTopic":true,"PinTopic":true,"CloseTopic":true}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (2,1,'{"ViewTopic":true,"CreateReply":true,"CloseTopic":true}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (3,1,'{}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (4,1,'{}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (5,1,'{}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (6,1,'{}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (1,2,'{"ViewTopic":true,"CreateReply":true,"CreateTopic":true,"LikeItem":true,"EditTopic":true,"DeleteTopic":true,"EditReply":true,"DeleteReply":true,"PinTopic":true,"CloseTopic":true}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (2,2,'{"ViewTopic":true,"CreateReply":true,"CreateTopic":true,"LikeItem":true,"EditTopic":true,"DeleteTopic":true,"EditReply":true,"DeleteReply":true,"PinTopic":true,"CloseTopic":true}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (3,2,'{"ViewTopic":true,"CreateReply":true,"CreateTopic":true,"LikeItem":true}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (4,2,'{"ViewTopic":true}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (5,2,'{"ViewTopic":true}');
INSERT INTO [forums_permissions] ([gid],[fid],[permissions]) VALUES (6,2,'{"ViewTopic":true}');
INSERT INTO [topics] ([title],[content],[parsed_content],[createdAt],[lastReplyAt],[lastReplyBy],[createdBy],[parentID],[ipaddress]) VALUES ('TestTopic','Atopicautomaticallygeneratedbythesoftware.','Atopicautomaticallygeneratedbythesoftware.',GETUTCDATE(),GETUTCDATE(),1,1,2,'::1');
INSERT INTO [replies] ([tid],[content],[parsed_content],[createdAt],[createdBy],[lastUpdated],[lastEdit],[lastEditBy],[ipaddress]) VALUES (1,'Areply!','Areply!',GETUTCDATE(),1,GETUTCDATE(),0,0,'::1');
