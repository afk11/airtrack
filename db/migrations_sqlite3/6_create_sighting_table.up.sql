create table `sighting` (
    `id` integer not null primary key autoincrement,
    `collection_site_id` int not null,
    `collection_session_id` int not null,
    `aircraft_id` int not null,
    `callsign` varchar(20) null,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `closed_at` timestamp null);
create unique index sighting_by_site on sighting(`collection_site_id`);