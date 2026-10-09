// Port of codec/decoder/core/inc/vlc_decoder.h.

package decoder

// SVlcTable (TagVlcTable). Each C `const uint8_t (*p)[2]` table pointer is a
// [][2]uint8 slice of the whole table (e.g. g_kuiVlcTable_0[:]).
type SVlcTable struct {
	kpCoeffTokenVlcTable       [4][8][][2]uint8
	kpChromaCoeffTokenVlcTable [][2]uint8
	kpZeroTable                [7][][2]uint8
	kpTotalZerosTable          [2][15][][2]uint8
}

// The g_kuiVlc*/g_kuiTotalZeros*/g_kuiZeroLeft* tables are defined in
// decoder_data_tables.go.

// WELS_GET_PREFIX_BITS(inval, outval) is `outval = GetPrefixBits(inval)`.
func WELS_GET_PREFIX_BITS(inval uint32) uint32 {
	return GetPrefixBits(inval)
}

func InitVlcTable(pVlcTable *SVlcTable) {
	pVlcTable.kpChromaCoeffTokenVlcTable = g_kuiVlcChromaTable[:]

	pVlcTable.kpCoeffTokenVlcTable[0][0] = g_kuiVlcTable_0[:]
	pVlcTable.kpCoeffTokenVlcTable[0][1] = g_kuiVlcTable_1[:]
	pVlcTable.kpCoeffTokenVlcTable[0][2] = g_kuiVlcTable_2[:]
	pVlcTable.kpCoeffTokenVlcTable[0][3] = g_kuiVlcTable_3[:]

	pVlcTable.kpCoeffTokenVlcTable[1][0] = g_kuiVlcTable_0_0[:]
	pVlcTable.kpCoeffTokenVlcTable[1][1] = g_kuiVlcTable_0_1[:]
	pVlcTable.kpCoeffTokenVlcTable[1][2] = g_kuiVlcTable_0_2[:]
	pVlcTable.kpCoeffTokenVlcTable[1][3] = g_kuiVlcTable_0_3[:]

	pVlcTable.kpCoeffTokenVlcTable[2][0] = g_kuiVlcTable_1_0[:]
	pVlcTable.kpCoeffTokenVlcTable[2][1] = g_kuiVlcTable_1_1[:]
	pVlcTable.kpCoeffTokenVlcTable[2][2] = g_kuiVlcTable_1_2[:]
	pVlcTable.kpCoeffTokenVlcTable[2][3] = g_kuiVlcTable_1_3[:]

	pVlcTable.kpCoeffTokenVlcTable[3][0] = g_kuiVlcTable_2_0[:]
	pVlcTable.kpCoeffTokenVlcTable[3][1] = g_kuiVlcTable_2_1[:]
	pVlcTable.kpCoeffTokenVlcTable[3][2] = g_kuiVlcTable_2_2[:]
	pVlcTable.kpCoeffTokenVlcTable[3][3] = g_kuiVlcTable_2_3[:]
	pVlcTable.kpCoeffTokenVlcTable[3][4] = g_kuiVlcTable_2_4[:]
	pVlcTable.kpCoeffTokenVlcTable[3][5] = g_kuiVlcTable_2_5[:]
	pVlcTable.kpCoeffTokenVlcTable[3][6] = g_kuiVlcTable_2_6[:]
	pVlcTable.kpCoeffTokenVlcTable[3][7] = g_kuiVlcTable_2_7[:]

	pVlcTable.kpZeroTable[0] = g_kuiZeroLeftTable0[:]
	pVlcTable.kpZeroTable[1] = g_kuiZeroLeftTable1[:]
	pVlcTable.kpZeroTable[2] = g_kuiZeroLeftTable2[:]
	pVlcTable.kpZeroTable[3] = g_kuiZeroLeftTable3[:]
	pVlcTable.kpZeroTable[4] = g_kuiZeroLeftTable4[:]
	pVlcTable.kpZeroTable[5] = g_kuiZeroLeftTable5[:]
	pVlcTable.kpZeroTable[6] = g_kuiZeroLeftTable6[:]

	pVlcTable.kpTotalZerosTable[0][0] = g_kuiTotalZerosTable0[:]
	pVlcTable.kpTotalZerosTable[0][1] = g_kuiTotalZerosTable1[:]
	pVlcTable.kpTotalZerosTable[0][2] = g_kuiTotalZerosTable2[:]
	pVlcTable.kpTotalZerosTable[0][3] = g_kuiTotalZerosTable3[:]
	pVlcTable.kpTotalZerosTable[0][4] = g_kuiTotalZerosTable4[:]
	pVlcTable.kpTotalZerosTable[0][5] = g_kuiTotalZerosTable5[:]
	pVlcTable.kpTotalZerosTable[0][6] = g_kuiTotalZerosTable6[:]
	pVlcTable.kpTotalZerosTable[0][7] = g_kuiTotalZerosTable7[:]
	pVlcTable.kpTotalZerosTable[0][8] = g_kuiTotalZerosTable8[:]
	pVlcTable.kpTotalZerosTable[0][9] = g_kuiTotalZerosTable9[:]
	pVlcTable.kpTotalZerosTable[0][10] = g_kuiTotalZerosTable10[:]
	pVlcTable.kpTotalZerosTable[0][11] = g_kuiTotalZerosTable11[:]
	pVlcTable.kpTotalZerosTable[0][12] = g_kuiTotalZerosTable12[:]
	pVlcTable.kpTotalZerosTable[0][13] = g_kuiTotalZerosTable13[:]
	pVlcTable.kpTotalZerosTable[0][14] = g_kuiTotalZerosTable14[:]
	pVlcTable.kpTotalZerosTable[1][0] = g_kuiTotalZerosChromaTable0[:]
	pVlcTable.kpTotalZerosTable[1][1] = g_kuiTotalZerosChromaTable1[:]
	pVlcTable.kpTotalZerosTable[1][2] = g_kuiTotalZerosChromaTable2[:]
}
