package migrations

import (
	"log"

	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

func init() {
	Register(Migration{
		Version: "v1.1.0",
		Name:    "OrganizationMember to Team + Package publisher_id + TeamPackage 去 permission",
		Execute: func(engine *xorm.Engine) error {
			if err := migrateOrgMemberToTeam(engine); err != nil {
				return err
			}
			return migrateTeamPackageRefine(engine)
		},
	})
}

func migrateOrgMemberToTeam(engine *xorm.Engine) error {
	var orgs []models.Organization
	if err := engine.Find(&orgs); err != nil {
		return err
	}

	if len(orgs) == 0 {
		log.Printf("[Migration v1.1.0] No organizations found, skipping")
		return nil
	}

	for _, org := range orgs {
		var members []models.OrganizationMember
		if err := engine.Where("organization_i_d = ?", org.ID).Find(&members); err != nil {
			return err
		}

		if len(members) == 0 {
			log.Printf("[Migration v1.1.0] Organization %s has no members, skipping", org.Name)
			continue
		}

		teamName := "team-" + org.Name
		var existingTeam models.Team
		has, err := engine.Where("name = ?", teamName).Get(&existingTeam)
		if err != nil {
			return err
		}

		var teamID int64
		if has {
			teamID = existingTeam.ID
			log.Printf("[Migration v1.1.0] Team %s already exists, using existing", teamName)
		} else {
			team := &models.Team{
				Name:        teamName,
				DisplayName: org.DisplayName + " 团队",
				Description: "从组织 " + org.Name + " 自动迁移",
				Permission:  "write",
			}
			if _, err := engine.Insert(team); err != nil {
				return err
			}
			teamID = team.ID
			log.Printf("[Migration v1.1.0] Created team %s (ID: %d)", teamName, teamID)
		}

		teamOrg := &models.TeamOrganization{
			TeamID:         teamID,
			OrganizationID: &org.ID,
		}
		if _, err := engine.Insert(teamOrg); err != nil {
			return err
		}
		log.Printf("[Migration v1.1.0] Linked team %s to organization %s", teamName, org.Name)

		for _, member := range members {
			var existingTeamMember models.TeamMember
			has, err := engine.Where("team_i_d = ? AND user_i_d = ?", teamID, member.UserID).Get(&existingTeamMember)
			if err != nil {
				return err
			}
			if has {
				log.Printf("[Migration v1.1.0] User %d already in team %s, skipping", member.UserID, teamName)
				continue
			}

			teamMember := &models.TeamMember{
				TeamID: teamID,
				UserID: member.UserID,
			}
			if _, err := engine.Insert(teamMember); err != nil {
				return err
			}
			log.Printf("[Migration v1.1.0] Added user %d to team %s", member.UserID, teamName)
		}

		log.Printf("[Migration v1.1.0] Migrated %d members from organization %s to team %s",
			len(members), org.Name, teamName)
	}

	log.Printf("[Migration v1.1.0] Migration completed. Old organization_members table preserved for rollback.")
	return nil
}

func migrateTeamPackageRefine(engine *xorm.Engine) error {
	if _, err := engine.Exec("ALTER TABLE packages ADD COLUMN publisher_i_d INTEGER DEFAULT 0"); err != nil {
		log.Printf("[Migration v1.1.0] 添加列 publisher_i_d: %v（可忽略）", err)
	} else {
		log.Println("[Migration v1.1.0] 添加列 publisher_i_d → OK")
	}

	if _, err := engine.Exec("ALTER TABLE team_packages DROP COLUMN permission"); err != nil {
		log.Printf("[Migration v1.1.0] 删除列 permission: %v（可忽略）", err)
	} else {
		log.Println("[Migration v1.1.0] 删除列 permission → OK")
	}

	log.Println("[Migration v1.1.0] 完成：Package.publisher_i_d + TeamPackage 去 permission")
	return nil
}