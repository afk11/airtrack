create table sighting_location (
    id serial not null primary key,
    sighting_id int not null,
    timestamp timestamp not null,
    altitude int not null,
    latitude numeric(12, 8) not null,
    longitude numeric(12, 8) not null);