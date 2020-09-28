create table `sighting` (
    `id` integer not null primary key autoincrement,
    `project_id` int not null,
    `session_id` int not null,
    `aircraft_id` int not null,
    `callsign` varchar(20) null,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `closed_at` timestamp null,
    `transmission_types` int unsigned not null default '0',
    `squawk` varchar(4) null);
create index sighting_closed_at on sighting(`project_id`);
create index sighting_project_id_aircraft_id_callsign_index on sighting(`project_id`, `aircraft_id`, `callsign`);
create index `sighting_aircraft_session` on `sighting`(`aircraft_id`,`session_id`);