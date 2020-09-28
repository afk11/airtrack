create table `sighting_kml` (
    `id` int unsigned not null auto_increment primary key,
    `sighting_id` int not null,
    `content_type` int not null,
    `kml` mediumblob not null) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `sighting_kml` add index `sighting_kml_sighting_id_index`(`sighting_id`);