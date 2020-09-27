create table `sighting` (
    `id` int unsigned not null auto_increment primary key,
    `project_id` int not null,
    `session_id` int not null,
    `aircraft_id` int not null,
    `callsign` varchar(20) null,
    `created_at` timestamp null,
    `updated_at` timestamp null,
    `closed_at` timestamp null,
    `squawk` varchar(4) null,
    `transmission_types` int unsigned not null default '0') default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `sighting` add index `closed_at`(`project_id`);
alter table `sighting` add index `sighting_project_id_aircraft_id_callsign_index`(`project_id`, `aircraft_id`, `callsign`);
ALTER TABLE `sighting`
    ADD INDEX `sighting_aircraft_session` (`aircraft_id`,`session_id`);