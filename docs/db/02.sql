-- changes for imap service
alter table message add uid int not null default 0;
alter table message add created_date datetime not null default current_timestamp;

alter table message add flag_seen varchar(1) not null default 'N';
alter table message add flag_answered varchar(1) not null default 'N';
alter table message add flag_flagged varchar(1) not null default 'N';
alter table message add flag_deleted varchar(1) not null default 'N';
alter table message add flag_draft varchar(1) not null default 'N';

alter table mailbox add flag_inbox varchar(1) not null default 'N';
alter table mailbox add flag_non_existent varchar(1) not null default 'N';
alter table mailbox add flag_no_inferiors varchar(1) not null default 'N';
alter table mailbox add flag_no_select varchar(1) not null default 'N';
alter table mailbox add flag_marked varchar(1) not null default 'N';
alter table mailbox add flag_subscribed varchar(1) not null default 'N';
alter table mailbox add flag_remote varchar(1) not null default 'N';
alter table mailbox add flag_archive varchar(1) not null default 'N';
alter table mailbox add flag_drafts varchar(1) not null default 'N';
alter table mailbox add flag_flagged varchar(1) not null default 'N';
alter table mailbox add flag_junk varchar(1) not null default 'N';
alter table mailbox add flag_sent varchar(1) not null default 'N';
alter table mailbox add flag_trash varchar(1) not null default 'N';
alter table mailbox add flag_important varchar(1) not null default 'N';

alter table mailbox drop column flag_inbox;
alter table message modify body longblob null;

-- add to config  this one  "imapAddress": "0.0.0.0:8993"

