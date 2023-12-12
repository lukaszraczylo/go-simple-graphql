package gql

import "testing"

func (suite *Tests) Test_searchForKeysInMapStringInterface() {
	type args struct {
		msi map[string]interface{}
		key string
	}
	tests := []struct {
		wantValue any
		args      args
		name      string
	}{
		{
			name: "Test_searchForKeysInMapStringInterface_exists",
			args: args{
				msi: map[string]interface{}{
					"gqlcache": true,
					"test":     "test2",
				},
				key: "test",
			},
			wantValue: "test2",
		},
		{
			name: "Test_searchForKeysInMapStringInterface_not_exists",
			args: args{
				msi: map[string]interface{}{
					"gqlcache": true,
					"test":     "test2",
				},
				key: "test-new",
			},
			wantValue: nil,
		},
	}
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			assert.Equal(tt.wantValue, searchForKeysInMapStringInterface(tt.args.msi, tt.args.key))
		})
	}
}
