-- add email header fields to message table
alter table message add message_date datetime;
alter table message add subject text;
alter table message add message_from text;
alter table message add sender text;
alter table message add reply_to text;
alter table message add message_to text;
alter table message add cc text;
alter table message add bcc text;
alter table message add in_reply_to text;
