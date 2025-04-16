ALTER TABLE pictures_moder_votes RENAME picture_moder_vote;
ALTER TABLE articles RENAME article;

DROP TABLE IF EXISTS pictures_types;
DROP TABLE IF EXISTS login_state;
DROP TABLE IF EXISTS acl_resources;
DROP TABLE IF EXISTS acl_resources_privileges;
DROP TABLE IF EXISTS acl_roles;
DROP TABLE IF EXISTS acl_roles_parents;
DROP TABLE IF EXISTS acl_roles_privileges_allowed;
DROP TABLE IF EXISTS acl_roles_privileges_denied;
