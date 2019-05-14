#!/bin/sh

test_description="sync overwrites modified files in detached head"

. ./lib/sharness.sh

# Create manifest repositories
manifest_url="file://${REPO_TEST_REPOSITORIES}/hello/manifests"

test_expect_success "setup" '
	# create .repo file as a barrier, not find .repo deeper
	touch .repo &&
	mkdir work
'

test_expect_success "git-repo sync to maint branch" '
	(
		cd work &&
		git-repo init -u $manifest_url -b maint &&
		git-repo sync
	)
'

test_expect_success "manifests version: 1.0" '
	(
		cd work &&
		cat >expect<<-EOF &&
		manifests: Version 1.0
		EOF
		git -C .repo/manifests log -1 --pretty="manifests: %s" >actual &&
		test_cmp expect actual
	)
'

test_expect_success "edit files in workdir, all projects are in detached HEAD" '
	(
		cd work &&
		test -f drivers/driver-1/VERSION &&
		echo hacked >drivers/driver-1/VERSION &&
		test -f projects/app1/VERSION &&
		echo hacked >projects/app1/VERSION &&
		test -f projects/app1/module1/VERSION &&
		echo hacked >projects/app1/module1/VERSION &&
		test -f projects/app2/VERSION &&
		echo hacked >projects/app2/VERSION
	)
'

test_expect_success "git-repo sync to master branch" '
	(
		cd work &&
		git-repo init -u $manifest_url -b master &&
		git-repo sync
	)
'

test_expect_success "manifests version: 2.0" '
	(
		cd work &&
		cat >expect<<-EOF &&
		manifests: Version 2.0
		EOF
		git -C .repo/manifests log -1 --pretty="manifests: %s" >actual &&
		test_cmp expect actual
	)
'

test_expect_success "projects synced in detached HEAD, changes will lost" '
	(
		cd work &&
		cat >expect<<-EOF &&
		driver-1: v2.0.0-dev
		app-1: v2.0.0-dev
		app-1.module1: v1.0.0
		app-2: v2.0.0-dev
		EOF
		echo "driver-1: $(cat drivers/driver-1/VERSION)" >actual &&
		echo "app-1: $(cat projects/app1/VERSION)" >>actual &&
		echo "app-1.module1: $(cat projects/app1/module1/VERSION)" >>actual &&
		echo "app-2: $(cat projects/app2/VERSION)" >>actual &&
		test_cmp expect actual
	)
'

test_done