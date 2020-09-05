create table sighting (
    id serial not null primary key,
    collection_site_id int not null,
    collection_session_id int not null,
    aircraft_id int not null,
    callsign varchar(20) null,
    created_at timestamp null,
    updated_at timestamp null,
    closed_at timestamp null);
create index sighting_closed_at on sighting(collection_site_id);