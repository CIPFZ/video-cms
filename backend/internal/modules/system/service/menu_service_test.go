package service

import (
	"testing"

	"github.com/CIPFZ/video-cms/internal/modules/system/model"
)

func TestBuildMenuTreePreservesGrandchildren(t *testing.T) {
	svc := &MenuService{}
	menus := []model.SysMenu{
		{BaseModel: model.SysMenu{}.BaseModel, ParentId: 0},
	}
	menus[0].ID = 1
	menus = append(menus,
		model.SysMenu{ParentId: 1},
		model.SysMenu{ParentId: 2},
	)
	menus[1].ID = 2
	menus[1].Name = "child"
	menus[2].ID = 3
	menus[2].Name = "grandchild"

	tree := svc.buildMenuTree(menus)
	if len(tree) != 1 {
		t.Fatalf("root count = %d, want 1", len(tree))
	}
	if len(tree[0].Children) != 1 {
		t.Fatalf("child count = %d, want 1", len(tree[0].Children))
	}
	if len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("grandchild count = %d, want 1", len(tree[0].Children[0].Children))
	}
	if tree[0].Children[0].Children[0].ID != 3 {
		t.Fatalf("grandchild id = %d, want 3", tree[0].Children[0].Children[0].ID)
	}
}
