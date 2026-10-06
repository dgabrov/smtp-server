- see where insert into message table is done
- you have body at your disposal
- using the emersion framework already used on the project, parse the header in a Header instance
- when inserting, please change the insert to actually insert into the following new fields:

message_date
subject
from
sender
reply_to
to
cc
bcc

Envelope from emersion has array of Address. When you have this, please convert this to string 
to the best of your ability (comma delimited values) and add the values as string. 
They will be used only for searching later. 

Please try to write the functionality for parsing the body to Envelope one 
time and use it across those. Please get directly strings rather than getting slice 
of Address and convert to string afterwards. Please write only one helper 
that converts to string and use it for all the fields where this is needed

