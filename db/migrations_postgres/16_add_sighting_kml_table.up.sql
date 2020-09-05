create table sighting_kml (
    id serial not null primary key,
    sighting_id int not null,
    content_type int not null,
    kml text not null);