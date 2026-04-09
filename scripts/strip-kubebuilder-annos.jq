walk(
  if type == "object" and has("description") and (.description | type) == "string"
  then
    if .description == "+kubebuilder:object:generate=true"
      then del(.description)
    else .description |= (split("\n") | map(select(test("^\\+kubebuilder") | not)) | join("\n") | rtrimstr("\n"))
    end
  else .
  end
)
