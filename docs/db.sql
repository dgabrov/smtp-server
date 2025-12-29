create table domain
(
    domain_id varchar(64)  not null
        primary key,
    name      varchar(255) not null,
    constraint ix_domain_name
        unique (name)
);

create table mailbox_user
(
    user_id   varchar(64)  not null
        primary key,
    domain_id varchar(64)  not null,
    login     varchar(255) not null,
    password  varchar(255) not null,
    constraint ix_mailbox_user
        unique (user_id, login),
    constraint fk_mailbox_user_domain
        foreign key (domain_id) references domain (domain_id)
);

create table mailbox
(
    mailbox_id varchar(64)  not null
        primary key,
    user_id    varchar(64)  not null,
    name       varchar(255) not null,
    constraint fk_mailbox_mailbox_user
        foreign key (user_id) references mailbox_user (user_id)
);

create table message
(
    message_id varchar(64) not null
        primary key,
    mailbox_id varchar(64) not null,
    body       longtext    null,
    constraint fk_message_mailbox
        foreign key (mailbox_id) references mailbox (mailbox_id)
);

