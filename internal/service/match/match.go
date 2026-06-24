package match

import (
	"sport_game2/internal/adapter/apifox"
)

type matchStore interface {
	GetMatchBetList(lotteryType, subType string) ([]apifox.MatchBetInfo, error)
	GetMatchBetInfo(lotteryType, matchCode string) (*apifox.MatchBetInfo, error)
	GetLastSpiderJobTime() (string, error)
}

type Service struct {
	repo matchStore
}

func NewService(repo matchStore) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetLastSpiderJobTime() (string, error) {
	return s.repo.GetLastSpiderJobTime()
}

func (s *Service) GetMatchBetList(lotteryType, subType, sortType string) (*apifox.GetMatchBetListReply, error) {
	list, err := s.repo.GetMatchBetList(lotteryType, subType)
	if err != nil {
		return nil, err
	}

	if sortType == "1" {
	}

	var groupsMap = make(map[string][]apifox.MatchBetInfo)
	var order []string
	for _, m := range list {
		issue := m.LotteryInfo.Issue
		if _, ok := groupsMap[issue]; !ok {
			order = append(order, issue)
		}
		groupsMap[issue] = append(groupsMap[issue], m)
	}

	var groups []apifox.MatchBetGroup
	for _, issue := range order {
		groups = append(groups, apifox.MatchBetGroup{
			Issue: issue,
			List:  groupsMap[issue],
		})
	}

	return &apifox.GetMatchBetListReply{
		List:   list,
		Groups: groups,
	}, nil
}

func (s *Service) GetMatchBetInfo(lotteryType, matchCode string) (*apifox.GetMatchBetInfoReply, error) {
	info, err := s.repo.GetMatchBetInfo(lotteryType, matchCode)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return &apifox.GetMatchBetInfoReply{Info: *info}, nil
}

func (s *Service) GetHotMatchList() (*apifox.GetHotMatchListReply, error) {
	list, err := s.repo.GetMatchBetList("227", "")
	if err != nil {
		return nil, err
	}

	var hotList []apifox.HotMatchInfo
	for _, m := range list {
		for _, bi := range m.LotteryInfo.BetInfos {
			if bi.SubType == "6" || bi.SubType == "1" {
				hotList = append(hotList, apifox.HotMatchInfo{
					MatchID:             m.MatchInfo.MatchID,
					MatchCnName:         m.TournamentInfo.CnName,
					MatchCnAlias:        m.TournamentInfo.CnAlias,
					AwayTeamName:        m.AwayTeamInfo.CnName,
					AwayTeamAliasName:   m.AwayTeamInfo.CnAlias,
					AwayTeamLogoURL:     m.AwayTeamInfo.LogoURL,
					HomeTeamName:        m.HomeTeamInfo.CnName,
					HomeTeamAliasName:   m.HomeTeamInfo.CnAlias,
					HomeTeamLogoURL:     m.HomeTeamInfo.LogoURL,
					LotteryRound:        m.LotteryInfo.Round,
					LotteryBetEndTimeStr: m.LotteryInfo.BetEndTimeStr,
					SubType:             bi.SubType,
					MatchCode:           m.LotteryInfo.MatchCode,
					Issue:               m.LotteryInfo.Issue,
					LotteryBetOptions:   bi.Options,
				})
				break
			}
		}
	}

	return &apifox.GetHotMatchListReply{
		List:  hotList,
		IsNew: true,
	}, nil
}
