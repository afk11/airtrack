create table `sighting_callsign` (`id` int unsigned not null auto_increment primary key, `sighting_id` int not null, `callsign` varchar(20) not null, `observed_at` timestamp not null) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `sighting_callsign` add index `sighting_callsign_sighting_id_index`(`sighting_id`);
alter table `sighting_callsign` add index `sighting_callsign_callsign_index`(`callsign`);
