create table `sighting_squawk` (`id` int unsigned not null auto_increment primary key, `sighting_id` int not null, `squawk` varchar(4) null, `observed_at` timestamp not null) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `sighting_squawk` add index `sighting_squawk_sighting_id_index`(`sighting_id`);
