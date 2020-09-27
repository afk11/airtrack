create table `sighting_kml` (
    `id` integer not null primary key autoincrement,
    `sighting_id` int not null,
    `content_type` int not null,
    `kml` blob not null
                            );
create index `sighting_kml_sighting_id_index` on `sighting_kml`(`sighting_id`);