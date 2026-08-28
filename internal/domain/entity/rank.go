package entity

// rank 표기 정규화.
//
// 클라이언트가 Plus 등급을 두 가지 표기로 섞어 보낸다 (2026-08-27 장애 원인).
// 서버 내부의 두 소비처가 요구하는 표기가 서로 다르기 때문에 한쪽으로 통일할 수 없다:
//
//   - records.rank (ent enum): "A_P" 만 허용. "A+" 는 RankValidator 에서 거부 → 500.
//   - t_rank."Rank" (EXP 계수 PK): "A+" 로 등록돼 있음. "A_P" 는 조회 미스 → EXP 미부여.
//
// 따라서 요청 원문을 보존한 채 각 소비처 직전에 그쪽 표기로 변환한다.
// t_rank 는 다른 서비스(wl_leaderboard 등)와 공유하는 인스턴스라 동결 상태다.
//
// 매핑 대상은 Plus 4종뿐이다. 그 외 등급(F/D/C/B/A/S/SS/SSS)은 양쪽 표기가 같다.
// 미지의 값은 변환하지 않고 그대로 통과시킨다 — 판단은 각 소비처에 맡긴다.

var rankPlusToUnderscore = map[string]string{
	"D+": "D_P",
	"C+": "C_P",
	"B+": "B_P",
	"A+": "A_P",
}

var rankUnderscoreToPlus = map[string]string{
	"D_P": "D+",
	"C_P": "C+",
	"B_P": "B+",
	"A_P": "A+",
}

// NormalizeRankForStorage 는 records.rank 저장용 표기('_P')로 변환한다.
// ent enum(RankValidator)이 허용하는 형태다.
func NormalizeRankForStorage(rank string) string {
	if v, ok := rankPlusToUnderscore[rank]; ok {
		return v
	}
	return rank
}

// NormalizeRankForCoefficient 는 t_rank 조회용 표기('+')로 변환한다.
// EXP 계수 테이블의 PK 형태다.
func NormalizeRankForCoefficient(rank string) string {
	if v, ok := rankUnderscoreToPlus[rank]; ok {
		return v
	}
	return rank
}
