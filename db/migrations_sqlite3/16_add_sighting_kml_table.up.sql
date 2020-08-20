create table `sighting_kml` (
    `id` integer not null primary key autoincrement,
    `sighting_id` int not null,
    `content_type` int not null,
    `kml` mediumtext not null
                            );