create table `collection_session` (`id` int unsigned not null auto_increment primary key, `identifier` varchar(100) not null, `collection_site_id` int not null, `created_at` timestamp null, `updated_at` timestamp null, `closed_at` timestamp null, `deleted_at` timestamp null) default character set utf8mb4 collate 'utf8mb4_unicode_ci';
alter table `collection_session` add index `closed_at`(`collection_site_id`);