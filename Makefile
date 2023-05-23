
.PHONY: age-plugin-tkey
age-plugin-tkey:
	go build ./cmd/age-plugin-tkey

clean:
	rm -f age-plugin-tkey
