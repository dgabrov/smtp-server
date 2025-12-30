alter table queue drop to_addr;
create table queue_recipient (
                                 queue_recipient_id varchar(64) not null,
                                 queue_id varchar(64) not null,
                                 to_addr varchar(255) not null,
                                 attempts int not null default 0,
                                 last_attempted_dt datetime
);

alter table queue_recipient add primary key (queue_recipient_id);
alter table queue_recipient add constraint fk_qrec_queue foreign key (queue_id) references queue(queue_id);


alter table queue_recipient add success_ind varchar(1) not null default 'N';
