create table `sighting_location` (
    `id` integer not null primary key autoincrement,
    `sighting_id` int not null,
    `timestamp` timestamp not null,
    `altitude` mediumint not null,
    `latitude` double(12, 8) not null,
    `longitude` double(12, 8) not null
                                 );