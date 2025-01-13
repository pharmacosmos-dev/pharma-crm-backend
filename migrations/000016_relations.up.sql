ALTER TABLE "orders" ADD FOREIGN KEY ("supplier_id") REFERENCES "suppliers" ("id");

ALTER TABLE "orders" ADD FOREIGN KEY ("store_id") REFERENCES "stores" ("id");

ALTER TABLE "orders" ADD FOREIGN KEY ("created_by") REFERENCES "employees" ("id");

ALTER TABLE "order_products" ADD FOREIGN KEY ("product_id") REFERENCES "products" ("id");

ALTER TABLE "order_products" ADD FOREIGN KEY ("order_id") REFERENCES "orders" ("id");

ALTER TABLE "order_products" ADD FOREIGN KEY ("created_by") REFERENCES "employees" ("id");

ALTER TABLE "customers" ADD FOREIGN KEY ("group_id") REFERENCES "groups" ("id");

ALTER TABLE "customers" ADD FOREIGN KEY ("tag_id") REFERENCES "tags" ("id");

ALTER TABLE "addresses" ADD FOREIGN KEY ("customer_id") REFERENCES "customers" ("id");