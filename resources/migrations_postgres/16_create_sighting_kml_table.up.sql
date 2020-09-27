create table sighting_kml (
    id serial not null primary key,
    sighting_id int not null,
    content_type int not null,
    kml bytea not null);
create index sighting_kml_sighting_id_index on sighting_kml(sighting_id);