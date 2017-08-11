ALTER TABLE ONLY tracker_items
    ADD COLUMN tracker_id bigint;

ALTER TABLE ONLY tracker_items
    ADD CONSTRAINT tracker_items_tracker_id_trackers_id_foreign 
        FOREIGN KEY (tracker_id)
        REFERENCES trackers(id)
        ON UPDATE RESTRICT 
        ON DELETE RESTRICT;

ALTER TABLE ONLY tracker_items
    ADD CONSTRAINT tracker_items_remote_item_id_tracker_id_uni_idx
        UNIQUE (remote_item_id, tracker_id);
