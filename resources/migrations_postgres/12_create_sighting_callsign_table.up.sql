create table sighting_callsign (
    id serial not null primary key,
    sighting_id int not null,
    callsign varchar(20) not null,
    observed_at timestamp not null);
create index sighting_callsign_sighting_id_index on sighting_callsign(sighting_id);
create index sighting_callsign_callsign_index on sighting_callsign(callsign);

