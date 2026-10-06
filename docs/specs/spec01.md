**Addl Database fields**
- database structure is in @docs/db/db_full.sql
- message table, write the script needed to add the following fields:


message_date time.Time
subject varchar(255)
from varchar(255)
sender varchar(255)
reply_to varchar(255)
to varchar(4000)
cc varcahr(4000)
bcc varchar(4000)
in_reply_to varchar(4000)

and deposit it in a new file called @docs/db/03.sql
