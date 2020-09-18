create table `junzis_registry` (`id` int unsigned not null auto_increment primary key, `created_at` timestamp null, `updated_at` timestamp null, `icao_hex` varchar(6) not null, `registration` varchar(12) not null, `aircraft_manufacturer` varchar(255) null, `aircraft_model` varchar(255) not null, `owner_name` varchar(255) null) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `junzis_registry` add unique `junzis_registry_icao_hex_unique`(`icao_hex`);
alter table `junzis_registry` add index `junzis_registry_registration_index`(`registration`);
