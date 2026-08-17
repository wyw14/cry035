INSERT INTO buildings(id,name,address) VALUES
 ('building-a','云梯中心 A 座','高新路 88 号'),
 ('building-b','云梯中心 B 座','高新路 90 号')
ON CONFLICT(id) DO NOTHING;

INSERT INTO equipment_models(id,manufacturer,name,category) VALUES
 ('model-e1','华升设备','HS-E800','elevator'),
 ('model-f1','华升设备','HS-F2000','freight_elevator'),
 ('model-l1','安达机械','AD-L500','lifting_device')
ON CONFLICT(id) DO NOTHING;

INSERT INTO equipment(id,code,name,building_id,model_id,location,responsible_unit,operating_status,version,created_at,updated_at) VALUES
 ('equipment-1','EL-A-01','A 座 1 号客梯','building-a','model-e1','A 座东厅','云梯物业工程部','running',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
 ('equipment-2','FE-B-01','B 座货梯','building-b','model-f1','B 座卸货区','迅达维保一组','running',1,'2026-03-01T00:00:00Z','2026-03-01T00:00:00Z')
ON CONFLICT(id) DO NOTHING;

INSERT INTO maintenance_programs(id,name,version,statutory_cycle_days,applicable_categories,checklist,active,updated_at) VALUES
 ('program-monthly','月度安全保养',1,30,ARRAY['elevator','freight_elevator'],
  '[{"id":"door-lock","name":"层门门锁联动","required":true,"evidence_required":true},{"id":"brake","name":"制动器间隙","unit":"mm","required":true,"maximum":0.7,"evidence_required":false},{"id":"lamp","name":"轿厢照明","required":false,"evidence_required":false}]'::jsonb,
  true,'2026-08-17T00:00:00Z'),
 ('program-quarterly-lift','升降设备季度检查',1,90,ARRAY['lifting_device'],
  '[{"id":"limit-switch","name":"限位开关","required":true,"evidence_required":true},{"id":"wire-rope","name":"钢丝绳磨损","required":true,"evidence_required":true}]'::jsonb,
  true,'2026-08-17T00:00:00Z')
ON CONFLICT(id) DO NOTHING;

