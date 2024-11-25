package resolvers

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.45

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/samber/lo"
	model "github.com/satont/twir/libs/gomodels"
	"github.com/satont/twir/libs/logger/audit"
	"github.com/satont/twir/libs/utils"
	"github.com/twirapp/twir/apps/api-gql/internal/gql/gqlmodel"
	"github.com/twirapp/twir/apps/api-gql/internal/gql/graph"
	"github.com/twirapp/twir/apps/api-gql/internal/gql/mappers"
	"gorm.io/gorm"
)

// TwitchCategories is the resolver for the twitchCategories field.
func (r *commandResponseResolver) TwitchCategories(
	ctx context.Context,
	obj *gqlmodel.CommandResponse,
) ([]gqlmodel.TwitchCategory, error) {
	var categories []gqlmodel.TwitchCategory

	for _, id := range obj.TwitchCategoriesIds {
		category, err := r.cachedTwitchClient.GetGame(ctx, id)
		if err != nil {
			r.logger.Error("failed to fetch twitch category", slog.Any("err", err))
			continue
		}
		if category == nil {
			continue
		}

		categories = append(
			categories,
			gqlmodel.TwitchCategory{
				ID:        id,
				Name:      category.Name,
				BoxArtURL: category.BoxArtURL,
			},
		)
	}

	return categories, nil
}

// CommandsCreate is the resolver for the commandsCreate field
func (r *mutationResolver) CommandsCreate(
	ctx context.Context,
	opts gqlmodel.CommandsCreateOpts,
) (*gqlmodel.CommandCreatePayload, error) {
	dashboardId, err := r.sessions.GetSelectedDashboard(ctx)
	if err != nil {
		return nil, err
	}

	user, err := r.sessions.GetAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}

	var createdCommands int64
	if err := r.gorm.
		WithContext(ctx).
		Model(&model.ChannelsCommands{}).
		Where(`"channelId" = ? AND "default" = ?`, dashboardId, false).
		Count(&createdCommands).
		Error; err != nil {
		return nil, fmt.Errorf("failed to count commands: %w", err)
	}

	if createdCommands >= 50 {
		return nil, fmt.Errorf("you can't create more than 50 custom commands")
	}

	if err := r.checkIsCommandWithNameOrAliaseExists(ctx, nil, opts.Name, opts.Aliases); err != nil {
		return nil, err
	}

	aliases := []string{}
	for _, alias := range opts.Aliases {
		a := strings.TrimSuffix(strings.ToLower(alias), "!")
		a = strings.ReplaceAll(a, " ", "")
		if a != "" {
			aliases = append(aliases, a)
		}
	}

	command := &model.ChannelsCommands{
		ID:                        uuid.New().String(),
		Name:                      strings.ToLower(opts.Name),
		Cooldown:                  null.IntFrom(int64(opts.Cooldown)),
		CooldownType:              opts.CooldownType,
		Enabled:                   opts.Enabled,
		Aliases:                   aliases,
		Description:               null.StringFrom(opts.Description),
		Visible:                   opts.Visible,
		ChannelID:                 dashboardId,
		Default:                   false,
		DefaultName:               null.String{},
		Module:                    "CUSTOM",
		IsReply:                   opts.IsReply,
		KeepResponsesOrder:        opts.KeepResponsesOrder,
		DeniedUsersIDS:            opts.DeniedUsersIds,
		AllowedUsersIDS:           opts.AllowedUsersIds,
		RolesIDS:                  opts.RolesIds,
		OnlineOnly:                opts.OnlineOnly,
		RequiredWatchTime:         opts.RequiredWatchTime,
		RequiredMessages:          opts.RequiredMessages,
		RequiredUsedChannelPoints: opts.RequiredUsedChannelPoints,
		Responses: make(
			[]*model.ChannelsCommandsResponses,
			0,
			len(opts.Responses),
		),
		GroupID:           null.StringFromPtr(opts.GroupID.Value()),
		CooldownRolesIDs:  opts.CooldownRolesIds,
		EnabledCategories: opts.EnabledCategories,
	}

	if opts.ExpiresAt.IsSet() && opts.ExpiresAt.Value() != nil {
		command.ExpiresAt = null.TimeFrom(time.UnixMilli(int64(*opts.ExpiresAt.Value())))
	}

	if opts.ExpiresType.IsSet() && opts.ExpiresType.Value() != nil {
		command.ExpiresType = lo.ToPtr(mappers.CommandsExpiresAtGqlToDb(*opts.ExpiresType.Value()))
	}

	for idx, res := range opts.Responses {
		if res.Text == "" {
			continue
		}

		command.Responses = append(
			command.Responses,
			&model.ChannelsCommandsResponses{
				ID:                uuid.New().String(),
				Text:              null.StringFrom(res.Text),
				Order:             idx,
				TwitchCategoryIDs: append(pq.StringArray{}, res.TwitchCategoriesIds...),
			},
		)
	}

	err = r.gorm.WithContext(ctx).Create(command).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create command: %w", err)
	}

	if err := r.cachedCommandsClient.Invalidate(ctx, dashboardId); err != nil {
		r.logger.Error("failed to invalidate commands cache", slog.Any("err", err))
	}

	r.logger.Audit(
		"New command created",
		audit.Fields{
			NewValue:      command,
			ActorID:       lo.ToPtr(user.ID),
			ChannelID:     lo.ToPtr(dashboardId),
			System:        mappers.AuditSystemToTableName(gqlmodel.AuditLogSystemChannelCommand),
			OperationType: audit.OperationCreate,
			ObjectID:      &command.ID,
		},
	)

	return &gqlmodel.CommandCreatePayload{ID: command.ID}, nil
}

// CommandsUpdate is the resolver for the commandsUpdate field.
func (r *mutationResolver) CommandsUpdate(
	ctx context.Context,
	id string,
	opts gqlmodel.CommandsUpdateOpts,
) (bool, error) {
	dashboardId, err := r.sessions.GetSelectedDashboard(ctx)
	if err != nil {
		return false, err
	}

	user, err := r.sessions.GetAuthenticatedUser(ctx)
	if err != nil {
		return false, err
	}

	if opts.Name.IsSet() {
		if err := r.checkIsCommandWithNameOrAliaseExists(
			ctx,
			&id,
			*opts.Name.Value(),
			nil,
		); err != nil {
			return false, err
		}
	}

	aliases := []string{}

	if opts.Aliases.IsSet() {
		if err := r.checkIsCommandWithNameOrAliaseExists(
			ctx,
			&id,
			"",
			opts.Aliases.Value(),
		); err != nil {
			return false, err
		}
	}

	cmd := &model.ChannelsCommands{}
	if err := r.gorm.
		WithContext(ctx).
		Where(
			`"id" = ? AND "channelId" = ?`, id, dashboardId,
		).
		First(cmd).
		Error; err != nil {
		return false, fmt.Errorf("command not found: %w", err)
	}

	if opts.ExpiresType.Value() != nil && *opts.ExpiresType.Value() == gqlmodel.CommandExpiresTypeDelete && cmd.Default {
		return false, fmt.Errorf("cannot delete at expire for default command")
	}

	var cmdCopy model.ChannelsCommands
	err = utils.DeepCopy(cmd, &cmdCopy)
	if err != nil {
		return false, fmt.Errorf("cannot create copy of command: %w", err)
	}

	if opts.Name.IsSet() {
		cmd.Name = strings.ToLower(*opts.Name.Value())
	}

	if opts.Cooldown.IsSet() {
		cmd.Cooldown = null.IntFrom(int64(*opts.Cooldown.Value()))
	}

	if opts.CooldownType.IsSet() {
		cmd.CooldownType = *opts.CooldownType.Value()
	}

	if opts.Enabled.IsSet() {
		cmd.Enabled = *opts.Enabled.Value()
	}

	if opts.Aliases.IsSet() {
		for _, alias := range opts.Aliases.Value() {
			a := strings.TrimSuffix(strings.ToLower(alias), "!")
			a = strings.ReplaceAll(a, " ", "")
			if a != "" {
				aliases = append(aliases, a)
			}
		}

		cmd.Aliases = aliases
	}

	if opts.Description.IsSet() {
		cmd.Description = null.StringFromPtr(opts.Description.Value())
	}

	if opts.Visible.IsSet() {
		cmd.Visible = *opts.Visible.Value()
	}

	if opts.IsReply.IsSet() {
		cmd.IsReply = *opts.IsReply.Value()
	}

	if opts.KeepResponsesOrder.IsSet() {
		cmd.KeepResponsesOrder = *opts.KeepResponsesOrder.Value()
	}

	if opts.AllowedUsersIds.IsSet() {
		cmd.AllowedUsersIDS = opts.AllowedUsersIds.Value()
	}

	if opts.DeniedUsersIds.IsSet() {
		cmd.DeniedUsersIDS = opts.DeniedUsersIds.Value()
	}

	if opts.RolesIds.IsSet() {
		cmd.RolesIDS = opts.RolesIds.Value()
	}

	if opts.OnlineOnly.IsSet() {
		cmd.OnlineOnly = *opts.OnlineOnly.Value()
	}

	if opts.RequiredWatchTime.IsSet() {
		cmd.RequiredWatchTime = *opts.RequiredWatchTime.Value()
	}

	if opts.RequiredMessages.IsSet() {
		cmd.RequiredMessages = *opts.RequiredMessages.Value()
	}

	if opts.RequiredUsedChannelPoints.IsSet() {
		cmd.RequiredUsedChannelPoints = *opts.RequiredUsedChannelPoints.Value()
	}

	if opts.GroupID.IsSet() {
		cmd.GroupID = null.StringFromPtr(opts.GroupID.Value())
	}

	if opts.CooldownRolesIds.IsSet() {
		cmd.CooldownRolesIDs = opts.CooldownRolesIds.Value()
	}

	if opts.EnabledCategories.IsSet() {
		cmd.EnabledCategories = opts.EnabledCategories.Value()
	}

	if opts.Responses.IsSet() {
		cmd.Responses = make([]*model.ChannelsCommandsResponses, 0, len(opts.Responses.Value()))
		for idx, res := range opts.Responses.Value() {
			if res.Text == "" {
				continue
			}

			response := &model.ChannelsCommandsResponses{
				Text:              null.StringFrom(res.Text),
				Order:             idx,
				CommandID:         cmd.ID,
				TwitchCategoryIDs: append(pq.StringArray{}, res.TwitchCategoriesIds...),
			}

			cmd.Responses = append(cmd.Responses, response)
		}
	}

	if opts.ExpiresType.IsSet() && opts.ExpiresAt.IsSet() {
		if opts.ExpiresType.Value() == nil {
			cmd.ExpiresType = nil
		} else {
			cmd.ExpiresType = lo.ToPtr(mappers.CommandsExpiresAtGqlToDb(*opts.ExpiresType.Value()))
		}

		if opts.ExpiresAt.Value() == nil {
			cmd.ExpiresAt = null.Time{}
		} else {
			cmd.ExpiresAt = null.TimeFrom(time.UnixMilli(int64(*opts.ExpiresAt.Value())))
		}
	}

	txErr := r.gorm.
		WithContext(ctx).
		Transaction(
			func(tx *gorm.DB) error {
				if opts.Responses.IsSet() {
					if err = tx.Delete(
						&model.ChannelsCommandsResponses{},
						`"commandId" = ?`,
						cmd.ID,
					).Error; err != nil {
						return err
					}
				}

				return tx.Save(cmd).Error
			},
		)
	if txErr != nil {
		return false, fmt.Errorf("failed to update command: %w", txErr)
	}

	if err := r.cachedCommandsClient.Invalidate(ctx, dashboardId); err != nil {
		r.logger.Error("failed to invalidate commands cache", slog.Any("err", err))
	}

	r.logger.Audit(
		"Command edited",
		audit.Fields{
			OldValue:      cmdCopy,
			NewValue:      cmd,
			ActorID:       lo.ToPtr(user.ID),
			ChannelID:     lo.ToPtr(dashboardId),
			System:        mappers.AuditSystemToTableName(gqlmodel.AuditLogSystemChannelCommand),
			OperationType: audit.OperationUpdate,
			ObjectID:      &cmd.ID,
		},
	)

	return true, nil
}

// CommandsRemove is the resolver for the commandsRemove field.
func (r *mutationResolver) CommandsRemove(ctx context.Context, id string) (bool, error) {
	dashboardId, err := r.sessions.GetSelectedDashboard(ctx)
	if err != nil {
		return false, err
	}

	user, err := r.sessions.GetAuthenticatedUser(ctx)
	if err != nil {
		return false, err
	}

	cmd := &model.ChannelsCommands{}

	if err := r.gorm.
		WithContext(ctx).
		Where(`"id" = ? AND "channelId" = ?`, id, dashboardId).
		First(cmd).
		Error; err != nil {
		return false, err
	}

	if cmd.Default {
		return false, fmt.Errorf("cannot remove default command")
	}

	if err := r.gorm.WithContext(ctx).Delete(&cmd).Error; err != nil {
		return false, err
	}

	if err := r.cachedCommandsClient.Invalidate(ctx, dashboardId); err != nil {
		r.logger.Error("failed to invalidate commands cache", slog.Any("err", err))
	}

	r.logger.Audit(
		"Command removed",
		audit.Fields{
			OldValue:      cmd,
			ActorID:       lo.ToPtr(user.ID),
			ChannelID:     lo.ToPtr(dashboardId),
			System:        mappers.AuditSystemToTableName(gqlmodel.AuditLogSystemChannelCommand),
			OperationType: audit.OperationDelete,
			ObjectID:      &cmd.ID,
		},
	)

	return true, nil
}

// Commands is the resolver for the commands field.
func (r *queryResolver) Commands(ctx context.Context) ([]gqlmodel.Command, error) {
	dashboardId, err := r.sessions.GetSelectedDashboard(ctx)
	if err != nil {
		return nil, err
	}

	var entities []model.ChannelsCommands
	if err := r.gorm.
		WithContext(ctx).
		Where(`"channelId" = ?`, dashboardId).
		Preload("Group").
		Preload("Responses").
		Order("name ASC").
		Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch commands: %w", err)
	}

	convertedCommands := make([]gqlmodel.Command, 0, len(entities))
	for _, entity := range entities {
		cooldown := entity.Cooldown.Int64
		cooldownInt := int(cooldown)
		responses := make([]gqlmodel.CommandResponse, 0, len(entity.Responses))

		slices.SortFunc(
			entity.Responses,
			func(a, b *model.ChannelsCommandsResponses) int {
				return cmp.Compare(a.Order, b.Order)
			},
		)

		for _, response := range entity.Responses {
			responses = append(
				responses,
				gqlmodel.CommandResponse{
					ID:                  response.ID,
					CommandID:           response.CommandID,
					Text:                response.Text.String,
					TwitchCategoriesIds: response.TwitchCategoryIDs,
					TwitchCategories:    make([]gqlmodel.TwitchCategory, 0, len(response.TwitchCategoryIDs)),
				},
			)
		}

		converted := gqlmodel.Command{
			ID:                        entity.ID,
			Name:                      entity.Name,
			Description:               entity.Description.String,
			Aliases:                   entity.Aliases,
			Cooldown:                  cooldownInt,
			CooldownType:              entity.CooldownType,
			Enabled:                   entity.Enabled,
			Visible:                   entity.Visible,
			Default:                   entity.Default,
			DefaultName:               entity.DefaultName.Ptr(),
			Module:                    entity.Module,
			IsReply:                   entity.IsReply,
			KeepResponsesOrder:        entity.KeepResponsesOrder,
			DeniedUsersIds:            entity.DeniedUsersIDS,
			AllowedUsersIds:           entity.AllowedUsersIDS,
			RolesIds:                  entity.RolesIDS,
			OnlineOnly:                entity.OnlineOnly,
			CooldownRolesIds:          entity.CooldownRolesIDs,
			EnabledCategories:         entity.EnabledCategories,
			RequiredWatchTime:         entity.RequiredWatchTime,
			RequiredMessages:          entity.RequiredMessages,
			RequiredUsedChannelPoints: entity.RequiredUsedChannelPoints,
			GroupID:                   entity.GroupID.Ptr(),
			Responses:                 responses,
		}

		if entity.ExpiresAt.Valid {
			converted.ExpiresAt = lo.ToPtr(int(entity.ExpiresAt.Time.UnixMilli()))
		}

		if entity.ExpiresType != nil {
			converted.ExpiresType = lo.ToPtr(mappers.CommandsExpiresAtDbToGql(*entity.ExpiresType))
		}

		if entity.Group != nil {
			converted.Group = &gqlmodel.CommandGroup{
				ID:    entity.Group.ID,
				Name:  entity.Group.Name,
				Color: entity.Group.Color,
			}
		}

		convertedCommands = append(convertedCommands, converted)
	}
	return convertedCommands, nil
}

// CommandsPublic is the resolver for the commandsPublic field.
func (r *queryResolver) CommandsPublic(
	ctx context.Context,
	channelID string,
) ([]gqlmodel.PublicCommand, error) {
	if channelID == "" {
		return nil, fmt.Errorf("channelID is required")
	}

	var entities []model.ChannelsCommands
	if err := r.gorm.
		WithContext(ctx).
		Where(`"channelId" = ? AND "visible" = true AND "enabled" = true`, channelID).
		Preload("Group").
		Preload("Responses").
		Order("name ASC").
		Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch commands: %w", err)
	}

	convertedCommands := make([]gqlmodel.PublicCommand, 0, len(entities))
	for _, entity := range entities {
		converted := gqlmodel.PublicCommand{
			Name:         entity.Name,
			Description:  entity.Description.String,
			Aliases:      entity.Aliases,
			Responses:    make([]string, 0, len(entity.Responses)),
			Cooldown:     int(entity.Cooldown.Int64),
			CooldownType: entity.CooldownType,
			Module:       entity.Module,
			Permissions:  make([]gqlmodel.PublicCommandPermission, 0),
		}

		for _, response := range entity.Responses {
			converted.Responses = append(converted.Responses, response.Text.String)
		}

		var roles []*model.ChannelRole
		if len(entity.RolesIDS) > 0 {
			ids := lo.Map(entity.RolesIDS, func(id string, _ int) string { return id })
			err := r.gorm.WithContext(ctx).
				Where(`"channelId" = ? AND "id" IN ?`, channelID, ids).
				Find(&roles).Error

			if err != nil {
				r.logger.Error("cannot get roles", slog.Any("err", err))
			} else {
				for _, role := range roles {
					converted.Permissions = append(
						converted.Permissions,
						gqlmodel.PublicCommandPermission{
							Name: role.Name,
							Type: role.Type.String(),
						},
					)
				}
			}
		}

		convertedCommands = append(convertedCommands, converted)
	}

	return convertedCommands, nil
}

// CommandResponse returns graph.CommandResponseResolver implementation.
func (r *Resolver) CommandResponse() graph.CommandResponseResolver {
	return &commandResponseResolver{r}
}

type commandResponseResolver struct{ *Resolver }
