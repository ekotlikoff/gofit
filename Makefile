sync_procreate_images:
	@echo "ACTION REQUIRED: export the procreate files as pngs to ~/Downloads/movements/"
	@read -p "  Are you ready to sync from ~/Downloads/movements/? [y/N]" -n 1 -r && [[ $$REPLY =~ ^[Yy] ]]
	./sync_pngs.sh

.PHONY: \
	sync_procreate_images \
