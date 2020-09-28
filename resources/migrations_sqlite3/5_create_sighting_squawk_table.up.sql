create table `sighting_squawk` (
    `id` integer not null primary key autoincrement,
    `sighting_id` int not null,
    `squawk` varchar(4) null,
    `observed_at` timestamp not null
                               );
create index sighting_squawk_sighting_id_index on sighting_squawk(`sighting_id`);
