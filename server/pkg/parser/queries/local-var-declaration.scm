; Variables declared at the top level of a function body
(func_definition
  body: (macro_func_body
    (compound_stmt
      (declaration_stmt (declaration) @local))))

; Variables in plain nested blocks: { { int x; } }
(func_definition
  body: (macro_func_body
    (compound_stmt
      (compound_stmt
        (declaration_stmt (declaration) @local)))))

; Variables declared inside nested blocks (1 level deep):
; while, for, foreach, if, do, defer, etc.
(func_definition
  body: (macro_func_body
    (compound_stmt
      (_
        (compound_stmt
          (declaration_stmt (declaration) @local))))))

; Variables in else blocks (1 level deep):
; else_part is a child of if_stmt, not matched by the _ > compound_stmt pattern
(func_definition
  body: (macro_func_body
    (compound_stmt
      (if_stmt
        (else_part
          body: (compound_stmt
            (declaration_stmt (declaration) @local)))))))

; Variables declared inside switch/case/default (1 level):
(func_definition
  body: (macro_func_body
    (compound_stmt
      (switch_stmt
        (switch_body
          (case_stmt
            (declaration_stmt (declaration) @local)))))))

(func_definition
  body: (macro_func_body
    (compound_stmt
      (switch_stmt
        (switch_body
          (default_stmt
            (declaration_stmt (declaration) @local)))))))

; Variables declared inside nested blocks (2 levels deep):
(func_definition
  body: (macro_func_body
    (compound_stmt
      (_
        (compound_stmt
          (_
            (compound_stmt
              (declaration_stmt (declaration) @local))))))))

; Variables in else blocks (2 levels deep):
(func_definition
  body: (macro_func_body
    (compound_stmt
      (_
        (compound_stmt
          (if_stmt
            (else_part
              body: (compound_stmt
                (declaration_stmt (declaration) @local)))))))))

; Variables inside switch within nested blocks (2 levels):
(func_definition
  body: (macro_func_body
    (compound_stmt
      (_
        (compound_stmt
          (switch_stmt
            (switch_body
              (case_stmt
                (declaration_stmt (declaration) @local)))))))))

(func_definition
  body: (macro_func_body
    (compound_stmt
      (_
        (compound_stmt
          (switch_stmt
            (switch_body
              (default_stmt
                (declaration_stmt (declaration) @local)))))))))

; Variables declared inside nested blocks (3 levels deep):
(func_definition
  body: (macro_func_body
    (compound_stmt
      (_
        (compound_stmt
          (_
            (compound_stmt
              (_
                (compound_stmt
                  (declaration_stmt (declaration) @local))))))))))
