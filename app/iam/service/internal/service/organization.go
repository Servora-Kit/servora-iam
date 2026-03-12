package service

import (
	"context"

	orgpb "github.com/Servora-Kit/servora/api/gen/go/organization/service/v1"
	paginationpb "github.com/Servora-Kit/servora/api/gen/go/pagination/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrganizationService struct {
	orgpb.UnimplementedOrganizationServiceServer

	uc *biz.OrganizationUsecase
}

func NewOrganizationService(uc *biz.OrganizationUsecase) *OrganizationService {
	return &OrganizationService{uc: uc}
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, req *orgpb.CreateOrganizationRequest) (*orgpb.CreateOrganizationResponse, error) {
	org, err := s.uc.Create(ctx, &entity.Organization{
		Name:        req.Name,
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		return nil, err
	}
	return &orgpb.CreateOrganizationResponse{Organization: orgToProto(org)}, nil
}

func (s *OrganizationService) GetOrganization(ctx context.Context, req *orgpb.GetOrganizationRequest) (*orgpb.GetOrganizationResponse, error) {
	org, err := s.uc.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &orgpb.GetOrganizationResponse{Organization: orgToProto(org)}, nil
}

func (s *OrganizationService) ListOrganizations(ctx context.Context, req *orgpb.ListOrganizationsRequest) (*orgpb.ListOrganizationsResponse, error) {
	page, pageSize := extractPagination(req.Pagination)
	orgs, total, err := s.uc.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	items := make([]*orgpb.OrganizationInfo, 0, len(orgs))
	for _, o := range orgs {
		items = append(items, orgToProto(o))
	}
	return &orgpb.ListOrganizationsResponse{
		Organizations: items,
		Pagination:    buildPaginationResp(total, page, pageSize),
	}, nil
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, req *orgpb.UpdateOrganizationRequest) (*orgpb.UpdateOrganizationResponse, error) {
	org, err := s.uc.Update(ctx, &entity.Organization{
		ID:          req.Id,
		Name:        req.Name,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		return nil, err
	}
	return &orgpb.UpdateOrganizationResponse{Organization: orgToProto(org)}, nil
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, req *orgpb.DeleteOrganizationRequest) (*orgpb.DeleteOrganizationResponse, error) {
	if err := s.uc.Delete(ctx, req.Id); err != nil {
		return nil, err
	}
	return &orgpb.DeleteOrganizationResponse{Success: true}, nil
}

func (s *OrganizationService) AddMember(ctx context.Context, req *orgpb.AddMemberRequest) (*orgpb.AddMemberResponse, error) {
	m, err := s.uc.AddMember(ctx, &entity.OrganizationMember{
		OrganizationID: req.OrganizationId,
		UserID:         req.UserId,
		Role:           req.Role,
	})
	if err != nil {
		return nil, err
	}
	return &orgpb.AddMemberResponse{Member: orgMemberToProto(m)}, nil
}

func (s *OrganizationService) RemoveMember(ctx context.Context, req *orgpb.RemoveMemberRequest) (*orgpb.RemoveMemberResponse, error) {
	if err := s.uc.RemoveMember(ctx, req.OrganizationId, req.UserId); err != nil {
		return nil, err
	}
	return &orgpb.RemoveMemberResponse{Success: true}, nil
}

func (s *OrganizationService) ListMembers(ctx context.Context, req *orgpb.ListMembersRequest) (*orgpb.ListMembersResponse, error) {
	page, pageSize := extractPagination(req.Pagination)
	members, total, err := s.uc.ListMembers(ctx, req.OrganizationId, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*orgpb.OrganizationMemberInfo, 0, len(members))
	for _, m := range members {
		items = append(items, orgMemberToProto(m))
	}
	return &orgpb.ListMembersResponse{
		Members:    items,
		Pagination: buildPaginationResp(total, page, pageSize),
	}, nil
}

func (s *OrganizationService) UpdateMemberRole(ctx context.Context, req *orgpb.UpdateMemberRoleRequest) (*orgpb.UpdateMemberRoleResponse, error) {
	m, err := s.uc.UpdateMemberRole(ctx, req.OrganizationId, req.UserId, req.Role)
	if err != nil {
		return nil, err
	}
	return &orgpb.UpdateMemberRoleResponse{Member: orgMemberToProto(m)}, nil
}

func orgToProto(o *entity.Organization) *orgpb.OrganizationInfo {
	return &orgpb.OrganizationInfo{
		Id:          o.ID,
		Name:        o.Name,
		Slug:        o.Slug,
		DisplayName: o.DisplayName,
		CreatedAt:   timestamppb.New(o.CreatedAt),
		UpdatedAt:   timestamppb.New(o.UpdatedAt),
	}
}

func orgMemberToProto(m *entity.OrganizationMember) *orgpb.OrganizationMemberInfo {
	return &orgpb.OrganizationMemberInfo{
		Id:             m.ID,
		OrganizationId: m.OrganizationID,
		UserId:         m.UserID,
		UserName:       m.UserName,
		UserEmail:      m.UserEmail,
		Role:           m.Role,
		CreatedAt:      timestamppb.New(m.CreatedAt),
	}
}

func extractPagination(p *paginationpb.PaginationRequest) (int32, int32) {
	page := int32(1)
	pageSize := int32(20)
	if p != nil {
		if pm := p.GetPage(); pm != nil {
			if pm.Page > 0 {
				page = pm.Page
			}
			if pm.PageSize > 0 {
				pageSize = pm.PageSize
			}
		}
	}
	return page, pageSize
}

func buildPaginationResp(total int64, page, pageSize int32) *paginationpb.PaginationResponse {
	return &paginationpb.PaginationResponse{
		Mode: &paginationpb.PaginationResponse_Page{
			Page: &paginationpb.PagePaginationResponse{
				Total:    total,
				Page:     page,
				PageSize: pageSize,
			},
		},
	}
}
